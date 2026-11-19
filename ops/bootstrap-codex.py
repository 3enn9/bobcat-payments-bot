"""Run on the VPS once, as root. Never prints credentials."""
import json, os, pathlib, secrets, subprocess, urllib.request
root = pathlib.Path('/root/bobcat-payments-bot')
def db(sql):
    p=subprocess.run(['docker','exec','-i','vnutri-telegram-bot-db-1','sh','-c','MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot -N bobcat'],input=sql,text=True,capture_output=True)
    if p.returncode: raise RuntimeError('MySQL provisioning failed')
    return p.stdout
app_env=dict(x.split('=',1) for x in json.loads(subprocess.check_output(['docker','inspect','bobcat-payments-bot-app-1']))[0]['Config']['Env'] if '=' in x)
app_path=root/'.env.codex';agent_path=root/'.env.codex-agent'
if app_path.exists() or agent_path.exists():
    raise RuntimeError('Configuration exists; inspect it rather than overwrite credentials')
password=secrets.token_hex(32); service=secrets.token_hex(32); webhook=secrets.token_hex(32)
allowed=['equipment','fuel_entries','garage_work_logs','incoming_payments','invoice_banks','invoice_buyer_emails','invoice_buyers','invoice_items','invoice_payment_allocations','invoice_suppliers','invoices','miniapp_requests','operations','rogatka_requests','salary','worker_cash_entries','worker_days_off','workers']
sql="CREATE USER 'codex_reader'@'%' IDENTIFIED BY '"+password+"';\n"
for table in allowed: sql+='GRANT SELECT ON `bobcat`.`'+table+"` TO 'codex_reader'@'%';\n"
db(sql)
def save(path,values):
    fd=os.open(path,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
    with os.fdopen(fd,'w') as f: f.write(''.join(k+'='+v+'\n' for k,v in values.items()))
save(app_path,dict(CODEX_ENABLED='false',CODEX_MAX_OWNER_ID='242575328',CODEX_AGENT_URL='http://codex-agent:8090',CODEX_SERVICE_TOKEN=service,MAX_WEBHOOK_SECRET=webhook))
save(agent_path,dict(CODEX_SERVICE_TOKEN=service,AGENT_DB_USER='codex_reader',AGENT_DB_PASS=password,AGENT_DB_ADDR=app_env['DB_HOST']+':'+app_env.get('DB_PORT','3306'),AGENT_DB_NAME='bobcat'))
# Update only the existing PaymentsBot webhook; do not add another recipient.
base='https://platform-api2.max.ru/subscriptions'
def api(method,payload=None):
    req=urllib.request.Request(base,data=json.dumps(payload).encode() if payload else None,headers={'Authorization':app_env['MAX_TOKEN'],'Content-Type':'application/json'},method=method)
    with urllib.request.urlopen(req,timeout=30) as r:return json.load(r)
subs=api('GET').get('subscriptions',[])
matching=[s for s in subs if s.get('url','').endswith('/max/webhook')]
if len(matching)!=1: raise RuntimeError('Expected exactly one existing /max/webhook subscription')
s=matching[0]; result=api('POST',dict(url=s['url'],update_types=s.get('update_types',[]),secret=webhook))
if not result.get('success'):raise RuntimeError('MAX subscription update failed')
print('Read-only DB grants, private configuration and MAX webhook secret provisioned.')
