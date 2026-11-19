"""Run on VPS as root. Restrict app DB account and provision project-only control."""
import json,os,pathlib,secrets,shutil,subprocess,tarfile,io
ROOT=pathlib.Path('/root/bobcat-payments-bot')
POLICY=ROOT/'data/codex-policy'
POLICY.mkdir(mode=0o700,parents=True,exist_ok=True)
def readenv(path):return dict(line.split('=',1) for line in path.read_text().splitlines() if '=' in line and not line.startswith('#'))
def save(path,values):
    path.write_text(''.join(k+'='+v+'\n' for k,v in values.items()));path.chmod(0o600)
def db(sql):
    p=subprocess.run(['docker','exec','-i','vnutri-telegram-bot-db-1','sh','-c','MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot -N bobcat'],input=sql,text=True,capture_output=True)
    if p.returncode:raise RuntimeError('DB account upgrade failed')
app=readenv(ROOT/'.env.codex');agent=readenv(ROOT/'.env.codex-agent')
password=secrets.token_hex(32)
app_password=app.get('DB_PASS',secrets.token_hex(32))
db("CREATE USER IF NOT EXISTS 'codex_project'@'%' IDENTIFIED BY '"+password+"'; ALTER USER 'codex_project'@'%' IDENTIFIED BY '"+password+"'; GRANT ALL PRIVILEGES ON bobcat.* TO 'codex_project'@'%'; CREATE USER IF NOT EXISTS 'paymentsbot_app'@'%' IDENTIFIED BY '"+app_password+"'; GRANT ALL PRIVILEGES ON bobcat.* TO 'paymentsbot_app'@'%';")
agent.update(AGENT_DB_USER='codex_project',AGENT_DB_PASS=password,CODEX_PROJECT_ADMIN='true')
app.update(DB_USER='paymentsbot_app',DB_PASS=app_password)
save(ROOT/'.env.codex-agent',agent);save(ROOT/'.env.codex',app)
# A private working tree without env files, git credentials or deployment authority.
workspace=ROOT/'data/codex-workspace';workspace.mkdir(parents=True,exist_ok=True)
if not any(workspace.iterdir()):
    archive=subprocess.check_output(['git','-C',str(ROOT),'archive','HEAD'])
    with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
        for member in tar:
            parts=pathlib.PurePosixPath(member.name).parts
            if any(p.startswith('.') or p in {'node_modules','dist','data'} for p in parts) or not (member.isdir() or member.isfile()):continue
            if not parts or '..' in parts:raise RuntimeError('unsafe archive path')
            if member.name in {'login.md'}:continue
            if member.isdir():(workspace/member.name).mkdir(parents=True,exist_ok=True)
            else:
                dst=workspace/member.name;dst.parent.mkdir(parents=True,exist_ok=True);dst.write_bytes(tar.extractfile(member).read())
    for base,dirs,files in os.walk(workspace):
        os.chown(base,1000,1000)
        for f in files:os.chown(pathlib.Path(base)/f,1000,1000)
control=ROOT/'data/codex-control';control.mkdir(parents=True,exist_ok=True);control.chmod(0o755)
# Freeze runtime/build policy OUTSIDE agent-writable files.
shutil.copyfile(ROOT/'ops/codex-control.py',POLICY/'control.py')
shutil.copyfile(ROOT/'Dockerfile',POLICY/'Dockerfile.app')
shutil.copyfile(ROOT/'miniapp/Dockerfile',POLICY/'Dockerfile.miniapp')
compose=(ROOT/'docker-compose.yml').read_text().replace('      - .env\n','      - '+str(ROOT/'.env')+'\n')
(POLICY/'compose.yml').write_text(compose)
overlay=(ROOT/'docker-compose.codex.yml').read_text().replace('      - .env\n','      - '+str(ROOT/'.env')+'\n').replace('      - .env.codex\n','      - '+str(ROOT/'.env.codex')+'\n').replace('env_file: .env.codex-agent','env_file: '+str(ROOT/'.env.codex-agent'))
(POLICY/'codex.yml').write_text(overlay)
shutil.copyfile(ROOT/'ops/paymentsbot-codex-control.service','/etc/systemd/system/paymentsbot-codex-control.service')
subprocess.run(['systemctl','daemon-reload'],check=True)
subprocess.run(['systemctl','enable','--now','paymentsbot-codex-control.service'],check=True)
if subprocess.run(['docker','buildx','inspect','paymentsbot-codex-builder'],capture_output=True).returncode != 0:
    subprocess.run(['docker','buildx','create','--name','paymentsbot-codex-builder','--driver','docker-container','--bootstrap'],check=True)
print('Project-only DB accounts, private workspace and scoped deployment broker installed.')
