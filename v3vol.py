#!/usr/bin/env python

import json
import os
import sys
import time
import requests
import subprocess
import shlex
import re
import argparse

V3IO_CONF_PATH = '/etc/v3io'
debug = False


def fail_and_exit(msg, **kwargs):
    failure_object = {'status': 'failure', 'message': msg}
    failure_object.update(**kwargs)
    error_json = json.dumps(failure_object)
    sys.exit(error_json)

def exit_successfully(**kwargs):
    success_object = {'status': 'success'}
    success_object.update(**kwargs)
    print json.dumps(success_object)
    sys.exit()



def run_command(command, cwd=None, quiet=False):
    print 'Running cmd: {0}'.format(command)

    pipes = subprocess.Popen(shlex.split(command),
                             cwd=cwd,
                             shell=False,
                             stdout=subprocess.PIPE,
                             stderr=subprocess.PIPE,
                             executable='/bin/bash')
    stdout, stderr = pipes.communicate()
    retcode = pipes.returncode

    if retcode:
        if quiet:
            print 'cmd failed quietly with retcode: {0}'.format(retcode)
        else:
            fail_and_exit('Command failed', command=command, stdout=stdout, stderr=stderr, retcode=retcode)

    return stdout, stderr, retcode


def is_mounted(mount):
    stdout, stderr, retcode = run_command('findmnt -n {0}'.format(mount), quiet=True)
    if retcode or stdout.split()[0] != mount:
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

    if r.status_code != 201:
        return False, "Error %d creating session %s" % (r.status_code, r.text)

    # Get cookie
    session_id = r.json()['data']['id']
    cookie = {'sid': session_id}
    return True, cookie


def cookie_to_headers(cookie):
    return {
        'Cookie': 'session=j:{0}'.format(json.dumps(cookie))
    } if cookie is not None else None


def list_containers(url, session_cookie):
    response = requests.get(url + '/api/containers', headers=cookie_to_headers(session_cookie))
    if response.status_code != 200:
        fail_and_exit('Error reading containers', status_code=response.status_code, response=response.text)

    container_names = []
    for container in response.json()['data']:
        container_names += [container['attributes']['name']]
    return container_names


def create_container(url, name, session_cookie):
    payload = {'data': {'type': 'container', 'attributes': {'name': name}}}
    response = requests.post(url + '/api/containers', json=payload, headers=cookie_to_headers(session_cookie))
    if response.status_code != requests.codes.created:
        fail_and_exit('Failed creating data container', container_name=name, status_code=response.status_code, response=response.text)
    return response.json()


def osmount(fuse_path, dataurl, path, container='', data_sid=None):
    if not is_mounted(path):
        stdout, stderr, retcode = run_command('mkdir -p %s' % path)

        session_arg = '-s {0}'.format(data_sid if data_sid is not None else '')
        print "session:", session_arg
        if container:
            container = '-a ' + container

        cmdstr = "nohup {0} -c {1} -m {2} -u on {3} {4} > /dev/null 2>&1 &".\
            format(fuse_path, dataurl, path, container, session_arg)

        debug_print(debug, cmdstr)
        os.system(cmdstr)
        for i in [1, 2, 4]:
            if is_mounted(path):
                break

            if i == 4:
                fail_and_exit('Failed to mount device. Failed to create fuse mount at {0}'.format(path))

            time.sleep(i)


def load_config(mount_dir):

    try:
        with open(V3IO_CONF_PATH + '/v3io.conf', 'r') as f:
            return None, json.loads(f.read())
    except Exception as exc:
        fail_and_exit('Failed to mount device {0} , Failed to open/read v3io conf at {1}'
                      .format(mount_dir, V3IO_CONF_PATH),
                      exc=str(exc))

def mount(mount_path, json_params, v3args):
    mount_path = os.path.abspath(mount_path)

    # load mount policy from json
    params = {}
    try:
        params = json.loads(json_params)
    except Exception as exc:
        fail_and_exit('Failed to mount device {0}'.format(mount_path), exc=str(exc))

    container_name = params.get('container','').strip()

    if container_name == '':
        fail_and_exit('Failed to mount device {0} , missing container name in {1}'.format(mount_path, json_params))

    cluster = params.get('cluster','default').strip()
    subpath = params.get('subpath','').strip()
    dedicate = params.get('dedicate','true').strip().lower()   # dedicated Fuse mount (vs shared)
    container_create = params.get('create','false').strip().lower()   # create container if doesnt exist
    username = params.get('username', '').strip()              # username for authentication
    username = params.get('kubernetes.io/secret/username', username).strip()    # username from secret
    password = params.get('password', '').strip()              # pw for authentication
    password = params.get('kubernetes.io/secret/password', password).strip()    # pw from secret

    if not len(username):
        fail_and_exit('Authentication details missing. Please provide username')
    if not len(password):
        fail_and_exit('Authentication details missing. Please provide password')

    # Get v3io configuration
    root_path = v3args['root_path']
    fuse_path = v3args['fuse_path']

    # TBD support for multi-cluster
    clusters = v3args['clusters'][0]

    api_url = cluster['api_url']
    data_url = cluster['data_url']

    # create control and data sessions
    success, ctrl_cookie = create_control_session(api_url, username, password)
    if not success:
        fail_and_exit('Failed to create control session {0}'.format(ctrl_cookie))
    success, data_cookie = create_data_session(api_url, username, password)
    if not success:
        fail_and_exit('Failed to create data session {0}'.format(data_cookie))

    data_sid = data_cookie['sid']

    # check if data container exist
    container_names = list_containers(api_url, ctrl_cookie)

    if container_name not in container_names:
        if container_create.lower() in ['true','yes','y']:
            _ = create_container(api_url, container_name, ctrl_cookie)

        else:
            fail_and_exit('Failed to mount device {0} , Data Container {1} doesnt exist'
                          .format(mount_path, container_name))

    # if we want a dedicated v3io connection
    if dedicate in ['true','yes','y']:
        osmount(fuse_path, data_url, mount_path, container_name, data_sid=data_sid)
        exit_successfully()

    # if shared fuse mount is not up, mount it
    v3_mount_path = os.path.join(root_path, cluster)
    osmount(fuse_path, data_url, v3_mount_path, data_sid=data_sid)
    container_path = os.path.join(v3_mount_path, container_name)

    # create subpath
    if subpath:
        container_path = os.path.join(container_path, subpath)
        run_command('mkdir -p {0}'.format(container_path))

    # mkdir
    run_command('mkdir -p {0}'.format(mount_path))

    # mount bind
    run_command('/bin/mount --bind "{0}" "{1}"'.format(container_path, mount_path))

    exit_successfully()


def unmount(mount_path, json_params=''):
    mount_path = os.path.abspath(mount_path)
    print "Unmounting: {0}".format(mount_path)

    if not is_mounted(mount_path):
        exit_successfully()

    retcode, stdout, stderr = run_command('umount "{0}"'.format(mount_path))
    if retcode:
        fail_and_exit('Failed to unmount {0}'.format(mount_path),
                      stdout=stdout,
                      stderr=stderr,
                      retcode=retcode)

    os.rmdir(mount_path)
    exit_successfully()


def debug_print(debug, txt):
    if debug:
        with open('/tmp/v3vol.log', 'a') as hs:
            hs.write(str(txt) + '\n')

def register_arguments():
    parser = argparse.ArgumentParser(prog='v3vol', add_help=True)
    sub_parser = parser.add_subparsers(dest='action')
    for command in ['init', 'list', 'config', 'attach', 'detach', 'mount', 'unmount', 'clear']:
        command_sub_parser = sub_parser.add_parser(command)
        if command == 'attach':
            command_sub_parser.add_argument('-jp', '--json-params', type=str)
        elif command == 'detach':
            command_sub_parser.add_argument('-md', '--mount-device', type=str)
        elif command == 'mount':
            command_sub_parser.add_argument('-md', '--mount-dir', type=str, required=True)
            command_sub_parser.add_argument('-jp', '--json-params', type=str)
        elif command == 'unmount':
            command_sub_parser.add_argument('-md', '--mount-dir', type=str, required=True)
        elif command == 'config':
            command_sub_parser.add_argument('-md', '--mount-dir', type=str, required=True)

    return parser


if __name__ == '__main__':
    parser = register_arguments()
    args = parser.parse_args()

    v3args = load_config()
    debug = v3args['debug']

    debug_print(debug, args)

    if args.action == 'mount':
        mount(args.mount_dir, args.json_params, v3args)
    elif args.action == 'unmount':
        unmount(args.mount_dir, args.json_params)
    elif args.action == 'attach':
        exit_successfully(device='/dev/null')
    elif args.action in ['detach', 'init']:
        exit_successfully()
    elif args.action == 'list':
        os.system('mount | grep v3io')
    elif args.action == 'clear':
        _, out, err = run_command('mount', quiet=False)

        for l in out.splitlines():
            m = re.match(r'^v3io.*on (.*) type', l, re.M | re.I)
            if m:
                unmount(m.group(1), '')
    elif args.action == 'config':
        load_config(args.mount_dir)
