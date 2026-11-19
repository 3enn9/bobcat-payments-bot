#!/usr/bin/env python3
"""Host-side PaymentsBot deployment broker. No caller-provided host commands/config."""
import hashlib, http.server, json, os, pathlib, shutil, socketserver, stat, subprocess, tempfile, threading, time, uuid
ROOT=pathlib.Path('/root/bobcat-payments-bot')
DATA=ROOT/'data/codex-control'
WORK=ROOT/'data/codex-workspace'
POLICY=ROOT/'data/codex-policy'
SERVICES={'app','miniapp','caddy'}
PREFIXES={'cmd','internal','transport','miniapp','sql','assets','certs'}
FILES={'go.mod','go.sum','README.md'}
lock=threading.Lock()
jobs={}

def allowed(parts):
    return bool(parts) and not any(p.startswith('.') or p in {'node_modules','dist','data'} for p in parts) and (parts[0] in PREFIXES or len(parts)==1 and parts[0] in FILES)

def snapshot():
    stage=pathlib.Path(tempfile.mkdtemp(prefix='build-',dir=DATA))
    total=0
    digest=hashlib.sha256()
    try:
        for base,dirs,files,fd in os.fwalk(WORK,follow_symlinks=False):
            rel=pathlib.Path(base).relative_to(WORK)
            dirs[:]=sorted(d for d in dirs if allowed((rel/d).parts))
            for name in sorted(files):
                path=rel/name
                if not allowed(path.parts):continue
                try: src=os.open(name,os.O_RDONLY|os.O_NOFOLLOW|os.O_NONBLOCK,dir_fd=fd)
                except OSError:raise ValueError('symlink or unreadable source: '+str(path))
                with os.fdopen(src,'rb') as f:
                    st=os.fstat(f.fileno())
                    if not stat.S_ISREG(st.st_mode) or st.st_size>32*1024*1024:raise ValueError('unsupported source file')
                    data=f.read(32*1024*1024+1)
                total+=len(data)
                if total>160*1024*1024:raise ValueError('source tree too large')
                dest=stage/path;dest.parent.mkdir(parents=True,exist_ok=True);dest.write_bytes(data)
                digest.update(str(path).encode()+b'\0'+data)
        return stage,digest.hexdigest()
    except Exception:
        shutil.rmtree(stage);raise

def compose(*args):
    return ['docker','compose','--project-name','bobcat-payments-bot','--project-directory',str(ROOT),'-f',str(POLICY/'compose.yml'),'-f',str(POLICY/'codex.yml'),*args]

def record(job):
    tmp=DATA/(job['id']+'.tmp');tmp.write_text(json.dumps(job));tmp.replace(DATA/(job['id']+'.json'))

def host_run(command, out, timeout):
    # A deployment gets its own systemd unit, independent of the requesting tool/session.
    run=['systemd-run','--quiet','--wait','--pipe','--collect','--unit=paymentsbot-codex-'+uuid.uuid4().hex,
         '--property=Type=exec','--property=RuntimeMaxSec='+str(timeout),
         '--setenv=PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/snap/bin','--',*command]
    subprocess.run(run,stdin=subprocess.DEVNULL,stdout=out,stderr=out,timeout=timeout+30,check=True,start_new_session=True)

def deploy(job,stage):
    with lock:
        try:
            service=job['service'];job['state']='running';record(job)
            with (DATA/(job['id']+'.log')).open('w') as out:
                if job['action']=='build':
                    context=stage if service=='app' else stage/'miniapp'
                    cmd=['docker','buildx','build','--builder','paymentsbot-codex-builder','--load',
                         '-f',str(POLICY/('Dockerfile.app' if service=='app' else 'Dockerfile.miniapp')),
                         '-t','bobcat-payments-bot-'+service,str(context)]
                    host_run(cmd,out,900)
                    # Publish only regular source files. Policy, env and host configuration stay immutable.
                    backup=DATA/(job['id']+'-backup');backup.mkdir()
                    tracked=subprocess.check_output(['git','-C',str(ROOT),'ls-files','-z']).decode().split('\0')
                    for name in tracked:
                        rel=pathlib.PurePosixPath(name)
                        if not allowed(rel.parts):continue
                        old=ROOT/name
                        if old.is_file() and not old.is_symlink():
                            saved=backup/name;saved.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(old,saved)
                            if not(stage/name).exists():old.unlink()
                    for base,dirs,files in os.walk(stage):
                        for name in files:
                            src=pathlib.Path(base)/name;rel=src.relative_to(stage);dst=ROOT/rel
                            if not dst.parent.resolve().is_relative_to(ROOT):raise ValueError('unsafe destination')
                            dst.parent.mkdir(parents=True,exist_ok=True)
                            if dst.is_symlink():raise ValueError('symlink destination')
                            shutil.copyfile(src,dst);dst.chmod(0o644)
                    host_run(compose('up','-d','--no-deps','--no-build','--pull','never',service),out,90)
                else:host_run(compose('restart',service),out,90)
            job['state']='done'
        except Exception as e:job.update(state='failed',error=str(e)[:1000])
        finally:
            job['finished']=time.time();record(job)
            if stage:shutil.rmtree(stage,ignore_errors=True)

class Handler(http.server.BaseHTTPRequestHandler):
    def log_message(self,*a):pass
    def do_POST(self):
        try:
            size=int(self.headers.get('Content-Length','0'))
            if not 0<size<=8192:raise ValueError('invalid body size')
            data=json.loads(self.rfile.read(size));action=data.get('action');service=data.get('service')
            if action=='status':
                jid=data.get('job_id','')
                if jid:
                    if not isinstance(jid,str) or len(jid)!=32 or any(c not in '0123456789abcdef' for c in jid):raise ValueError('invalid job id')
                    file=DATA/(jid+'.json');result=json.loads(file.read_text())
                    log=DATA/(jid+'.log')
                    if log.exists():result['log']=log.read_text(errors='replace')[-16000:]
                else:
                    p=subprocess.run(compose('ps','--format','json'),capture_output=True,text=True,timeout=15,check=True)
                    result={'services':p.stdout}
            elif action in {'build','restart'} and service in SERVICES and not(action=='build' and service=='caddy'):
                stage,digest=snapshot() if action=='build' else (None,'')
                # Repeated requests during a deployment return its existing job.
                old=next((j for j in jobs.values() if j['state'] in {'queued','running'} and j['service']==service),None)
                if old:
                    if stage:shutil.rmtree(stage)
                    result=old
                else:
                    job={'id':uuid.uuid4().hex,'action':action,'service':service,'state':'queued','digest':digest,'created':time.time()}
                    jobs[job['id']]=job;record(job)
                    threading.Thread(target=deploy,args=(job,stage),daemon=True).start();result=job
            else:raise ValueError('only app/miniapp builds and app/miniapp/caddy restarts allowed')
            body=json.dumps(result).encode();self.send_response(200)
        except Exception as e:body=json.dumps({'error':str(e)}).encode();self.send_response(400)
        self.send_header('Content-Type','application/json');self.send_header('Content-Length',str(len(body)));self.end_headers();self.wfile.write(body)

if __name__=='__main__':
    DATA.mkdir(parents=True,exist_ok=True)
    sock=DATA/'control.sock'
    if sock.exists():sock.unlink()
    class Server(socketserver.ThreadingMixIn,socketserver.UnixStreamServer):daemon_threads=True
    with Server(str(sock),Handler) as server:
        os.chown(sock,0,1000);os.chmod(sock,0o660);server.serve_forever()
