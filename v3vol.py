#!/usr/bin/python

import json, os, sys, time, requests, envoy, re

V3IO_CONF_PATH = '/etc/v3io'
debug = False

base_config = """{
   "version": "1.0",
   "root_path": "/tmp/v3io",
   "fuse_path": "/home/iguazio/igz/clients/fuse/bin/v3io_adapters_fuse",
   "debug": false,
   "clusters": [
        {
                "name": "default",
                "data_url": "tcp://%s:1234",
                "api_url": "http://%s:4001"
        }
    ]
}"""

def perr(msg):
    txt = '{ "status": "Failure", "message": "%s"}' % msg
    # print txt
    sys.exit(txt)

def docmd(txt):
    cmd = '/bin/bash -c "{0}"'.format(txt)

    #p = Popen(txt.split(),stdout=PIPE)
    #sout, serr = p.communicate()
    #return p.returncode, sout, serr

    r = envoy.run(cmd)
    return r.status_code, r.std_out, r.std_err

def ismounted(mnt):
    ecode, sout, serr = docmd('findmnt -n %s' % mnt)
    if ecode or sout.split()[0]<>mnt:
        return False
    return True

def create_control_session(url, username, password):
    return create_session(url, username, password, 'control')

def create_data_session(url, username, password):
    return create_session(url, username, password, 'data')

def create_session(url, username, password, session_type='control'):
    payload = {
        'data': {
            'type': 'session',
            'attributes': {
                'plane': session_type,
                'interface_kind': 'fuse',
                'username': username,
                'password': password,
            }
        }
    }

    r = requests.post(url + '/api/sessions', json=payload)

    if r.status_code <> 201:
        return 1, "Error %d creating session %s" % (r.status_code, r.text)

    # Get cookie
    session_id = r.json()['data']['id']
    cookie = {'sid': session_id}
    return 0, cookie

def cookie_to_headers(cookie):
    return {
        'Cookie': 'session=j:' + json.dumps(cookie)
    } if cookie is not None else None

def list_containers(url, session_cookie):
    r = requests.get(url + '/api/containers', headers=cookie_to_headers(session_cookie))
    if r.status_code <> 200 :
        return 1,"Error %d reading containers %s" % (r.status_code,r.text)
    clist= []
    for c in r.json()['data'] :
        clist += [c['attributes']['name']]
    return 0, clist

def create_container(url, name, session_cookie):
    payload = {'data': {'type': 'container', 'attributes': {'name': name}}}
    r = requests.post(url + '/api/containers', json=payload, headers=cookie_to_headers(session_cookie))
    if r.status_code != requests.codes.created:
        return r.status_code,'failed creating container name:%s, reason:%s %s' % (name, r.status_code, r.content)
    return 0, eval(r.content)

def usage():
    print 'Failed to execute , usage:\n'
    print '  init'
    print '  list'
    print '  attach <json params>'
    print '  detach <mount device>'
    print '  mount <mount dir> [<mount device>] <json params>'
    print '  unmount <mount dir>'
    print '  config  <v3io IP address>'
    print '  clear\n'
    print ' Example: v3io mount /tmp/mymnt {"container":"datalake"}\n'
    sys.exit(1)

def osmount(fuse_path,dataurl,path,cnt='', data_sid=None):
    if not ismounted(path):
        ecode, sout, serr = docmd('mkdir -p %s' % path)

        session_arg = '-s %s' % (data_sid) if data_sid is not None else ''
        if cnt: cnt = '-a ' + cnt
        os.system("nohup %s -c %s -m %s -u on %s %s > /dev/null 2>&1 &"
                  % (fuse_path, dataurl, path, cnt, session_arg))
        for i in [1,2,4]:
            time.sleep(i)
            if ismounted(path): break
            if i == 4:
                perr('Failed to mount device , didnt manage to create fuse mount at %s' % (path))

def mount(args):
    mntpath = args[1]
    if len(args) == 4 :
        conf = args[3]
    else:
        conf = args[2]

    # load mount policy from json
    try :
        js = json.loads(conf)
    except :
            perr('Failed to mount device %s , bad json %s' % (mntpath,args[2]))
    cnt = js.get('container','').strip()

    if cnt == '' :
            perr('Failed to mount device %s , missing container name in %s' % (mntpath,args[2]))

    cluster = js.get('cluster','default').strip()
    subpath = js.get('subpath','').strip()
    dedicate = js.get('dedicate','true').strip().lower()   # dedicated Fuse mount (vs shared)
    createnew = js.get('create','false').strip().lower()   # create container if doesnt exist
    username = js.get('username', '').strip()              # username for authentication
    username = js.get('kubernetes.io/secret/username', username).strip()    # username from secret
    password = js.get('password', '').strip()              # pw for authentication
    password = js.get('kubernetes.io/secret/password', password).strip()    # pw from secret

    if not len(username):
        perr('Authentication details missing. Please provide username')
    if not len(password):
        perr('Authentication details missing. Please provide password')

    # Load v3io configuration
    try:
        f=open(V3IO_CONF_PATH+'/v3io.conf','r')
        v3args = json.loads(f.read())
        root_path = v3args['root_path']
        fuse_path = v3args['fuse_path']
        debug = v3args['debug']
        cl = v3args['clusters'][0]  #TBD support for multi-cluster
        apiurl = cl['api_url']
        dataurl = cl['data_url']
    except Exception,err:
        perr('Failed to mount device %s , Failed to open/read v3io conf at %s (%s)' % (mntpath,V3IO_CONF_PATH,err))

    # create control and data sessions
    e, ctrl_cookie = create_control_session(apiurl, username, password)
    if e: perr('Failed to create control session %s' % (ctrl_cookie))
    e, data_cookie = create_data_session(apiurl, username, password)
    if e: perr('Failed to create data session %s' % (data_cookie))

    data_sid = data_cookie['sid']

    # check if data container exist
    e, lc = list_containers(apiurl, ctrl_cookie)
    if e : perr(lc)
    if cnt not in lc :
        if createnew in ['true','yes','y'] :
            e, data = create_container(apiurl, cnt, ctrl_cookie)
            if e : perr('Failed to mount device %s , cant create Data Container %s (%s)' % (mntpath,cnt,data))
        else :
            perr('Failed to mount device %s , Data Container %s doesnt exist' % (mntpath,cnt))

    # if we want a dedicated v3io connection
    if dedicate in ['true','yes','y']:
        osmount(fuse_path, dataurl, mntpath, cnt, data_sid=data_sid)
        print '{"status": "Success"}'
        sys.exit()

    #if not os.path.isdir(cpath) :

    # if shared fuse mount is not up, mount it
    v3mpath = '/'.join([root_path,cluster])
    osmount(fuse_path,dataurl,v3mpath, data_sid=data_sid)
    cpath = '/'.join([v3mpath,cnt])

    # create subpath
    if subpath:
        cpath = '/'.join([cpath,subpath])
        ecode, sout, serr = docmd('mkdir -p %s' % cpath)
        if ecode :
            perr('Failed to create subpath %s under container %s, %s, %s' % (subpath,cnt,sout,serr))

    # mkdir
    ecode, sout, serr = docmd('mkdir -p %s' % mntpath)
    if ecode :
        perr('Failed to create mount dir %s %s %s' % (mntpath,sout,serr))

    # mount bind
    cmd = "/bin/mount --bind '%s' '%s'" % (cpath,mntpath)
    ecode, sout, serr = docmd(cmd)
    if ecode :
        perr('Failed to bind mount dir %s to %s, %s, %s' % (cpath,mntpath,sout,serr))

    print '{"status": "Success"}'

def unmount(args):
    mntpath = args[1]
    if mntpath[-1:]=='/' : mntpath=mntpath[:-1]  # remove trailing /

    if not ismounted(mntpath):
        print '{"status": "Success"}'
        sys.exit()

    ecode, sout, serr = docmd('umount "%s"' % mntpath)
    if ecode :
        perr('Failed to unmount %s , %s, %s' % (mntpath,sout,serr))

    os.rmdir(mntpath)
    print '{"status": "Success"}'


if __name__ == '__main__':
    args = sys.argv
    if len(args) < 2 : usage()
    cmd = args[1].lower()
    if cmd in ['mount','unmount','config'] and len(args) < 3 : usage()

    if   cmd=='mount' :
        mount(args[1:])
    elif cmd=='unmount'  :
        unmount(args[1:])
        sys.exit()
    elif cmd=='attach'  :
        print '{"status": "Success", "device": "/dev/null"}'
    elif cmd=='detach' or cmd=='init':
        print '{"status": "Success"}'
    elif cmd=='list':
        os.system('mount | grep v3io')
    elif cmd=='config':
        ecode, sout, serr = docmd('mkdir -p %s' % V3IO_CONF_PATH)
        f=open(V3IO_CONF_PATH+'/v3io.conf','w')
        f.write(base_config % (args[2],args[2]))
        f.close()
    elif cmd=='clear':
        ecode, sout, serr = docmd('mount')
        lines = sout.splitlines()
        for l in lines :
            m = re.match( r'^v3io.*on (.*) type', l, re.M|re.I)
            if m:
                print "Unmount: ",m.group(1),
                unmount(['',m.group(1)])
        sys.exit()
    else :
        usage()


