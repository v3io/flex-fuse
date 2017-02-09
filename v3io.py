#!/usr/bin/python

import json, os, sys, time, requests, envoy
from subprocess import Popen, PIPE

jj = '{"container":"vol1","kubernetes.io/fsType":"","kubernetes.io/readwrite":"rw","kubernetes.io/secret/password":"MWYyZDFlMmU2N2Rm","kubernetes.io/secret/username":"YWRtaW4=","url":"tcp://192.168.1.1"}'
V3IO_ROOT_PATH = '/tmp/v3io'
V3IO_FUSE_PATH = '/home/iguazio/igz/clients/fuse/bin/v3io_adapters_fuse'
V3IO_URL = 'tcp://192.168.154.57:1234'
V3IO_SERVICE_URL = 'http://192.168.154.57:4001'

def err(msg):
    txt = '{ "status": "Failure", "message": "%s"}' % msg
    # print txt
    sys.exit(txt)

def docmd(txt):
    # p = Popen(txt.split(),stdout=PIPE)
    # sout, serr = p.communicate()
    # return p.returncode, sout, serr

    cmd = '/bin/bash -c "{0}"'.format(txt)
    # print "exec: ", cmd
    r = envoy.run(cmd)

    return r.status_code, r.std_out, r.std_err

def ismounted(mnt):
    ecode, sout, serr = docmd('findmnt -n %s' % mnt)
    if ecode or sout.split()[0]<>mnt:
        return False
    return True

def list_containers():
    r = requests.get(V3IO_SERVICE_URL + '/api/containers')
    if r.status_code <> 200 :
        return 1,"Error %d reading containers %s" % (r.status_code,r.text)
    clist= []
    for c in r.json()['data'] :
        clist += [c['attributes']['name']]
    return 0, clist

def create_container(name):
    payload = {'data': {'type': 'container', 'attributes': {'name': name}}}
    r = requests.post(V3IO_SERVICE_URL + '/api/containers', json=payload)
    if r.status_code != r.codes.created:
        return r.status_code,'failed creating container name:%s, reason:%s %s' % (name, r.status_code, r.content)
    return 0, eval(r.content)

def usage():
    print 'Failed to mount device , only %d parameters, usage:\n' % len(args)
    print '  init'
    print '  list'
    print '  attach <json params>'
    print '  detach <mount device>'
    print '  mount <mount dir> <mount device> <json params>'
    print '  unmount <mount dir>'
    sys.exit(1)


def mount(args):
    if len(args) < 3 :
        err('Failed to mount device , only %d parameters, usage mount <mntpath> <json-params>' % len(args))
    mntpath = args[1]
    try :
        js = json.loads(args[2])
    except :
            err('Failed to mount device %s , bad json %s' % (mntpath,args[2]))
    cnt = js.get('container','').strip()
    if cnt == '' :
            err('Failed to mount device %s , missing container name in %s' % (mntpath,args[2]))

    cluster = js.get('cluster','default').strip()
    subpath = js.get('subpath','').strip()
    dedicate = js.get('dedicate','false').strip().lower()  # dedicated Fuse mount (vs shared)
    createnew = js.get('create','false').strip().lower()   # create container if doesnt exist

    # check if countainer exist
    e, lc = list_containers()
    if e : err(lc)
    if cnt not in lc :
        if createnew in ['true','yes','y'] :
            e, data = create_container(cnt)
            if e : err('Failed to mount device %s , cant create Data Container %s (%s)' % (mntpath,cnt,data))
        else :
            err('Failed to mount device %s , Data Container %s doesnt exist' % (mntpath,cnt))

    #if not os.path.isdir(cpath) :

    # if fuse not up mount
    v3mpath = '/'.join([V3IO_ROOT_PATH,cluster])
    if not ismounted(v3mpath):
        ecode, sout, serr = docmd('mkdir -p %s' % v3mpath)
        os.system("%s -b 16 -c %s -m %s -u on &" % (V3IO_FUSE_PATH,V3IO_URL,v3mpath))
        time.sleep(5)
        ecode, sout, serr = docmd('findmnt -n %s' % v3mpath)
        if ecode or sout.split()[0]<>v3mpath:
            err('Failed to mount device %s , didnt manage to create fuse mount at %s, %s %s' % (mntpath,v3mpath,sout, serr))

    cpath = '/'.join([v3mpath,cnt])

    # create subpath
    if subpath:
        cpath = '/'.join([cpath,subpath])
        ecode, sout, serr = docmd('mkdir -p %s' % cpath)
        if ecode :
            err('Failed to create subpath %s under container %s, %s, %s' % (subpath,cnt,sout,serr))

    # mkdir
    ecode, sout, serr = docmd('mkdir -p %s' % mntpath)
    if ecode :
        err('Failed to create mount dir %s %s %s' % (mntpath,sout,serr))

    # mount bind
    cmd = "/bin/mount --bind '%s' '%s'" % (cpath,mntpath)

    print cmd
    ecode, sout, serr = docmd(cmd)
    if ecode :
        err('Failed to bind mount dir %s to %s, %s, %s' % (cpath,mntpath,sout,serr))

    print '{"status": "Success"}'

def unmount(args):
    if len(args) < 2 :
        err('Failed to unmount device , only %d parameters, usage unmount <mntpath> ' % len(args))
    mntpath = args[1]
    if not ismounted(mntpath):
        print '{"status": "Success"xxx}'
        sys.exit()

    ecode, sout, serr = docmd('umount "%s"' % mntpath)
    if ecode :
        err('Failed to unmount %s , %s, %s' % [mntpath,sout,serr])

    os.rmdir(mntpath)
    print '{"status": "Success"}'
    sys.exit()


if __name__ == '__main__':
    args = sys.argv
    if len(args) < 2 :
        usage()

    cmd = args[1].lower()
    if   cmd=='mount' :
        mount(args[1:])
    elif cmd=='unmount'  :
        unmount(args[1:])
    elif cmd=='attach'  :
        print '{"status": "Success", "device": "/dev/null"}'
    elif cmd=='detach' or cmd=='init':
        print '{"status": "Success"}'
    elif cmd=='list':
        os.system('mount | grep v3io')
    else :
        usage()


