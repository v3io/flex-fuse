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
DEBUG = False



def debug_print(txt):
    if DEBUG:
        with open('/tmp/v3vol.log', 'a') as hs:
            hs.write(str(txt) + '\n')


def fail_and_exit(msg, **kwargs):
    failure_object = {'status': 'Failure', 'message': msg}
    failure_object.update(**kwargs)
    error_json = json.dumps(failure_object)
    sys.exit(error_json)


def exit_successfully(**kwargs):
    success_object = {'status': 'Success'}
    success_object.update(**kwargs)
    print json.dumps(success_object)
    sys.exit()


def run_command(command, cwd=None, quiet=False):
    debug_print('Running cmd: {0}'.format(command))

    pipes = subprocess.Popen(shlex.split(command),
                             cwd=cwd,
                             shell=False,
                             stdout=subprocess.PIPE,
                             stderr=subprocess.PIPE)
    stdout, stderr = pipes.communicate()
    retcode = pipes.returncode

    if retcode:
        if quiet:
            debug_print('Command failed quietly. stdout: {0}, stderr: {1}, retcode: {2}'.
                        format(stdout, stderr, retcode))
        else:
            fail_and_exit('Command failed', command=command, stdout=stdout, stderr=stderr, retcode=retcode)
    else:
        debug_print('Command ran successfully. stdout: {0}, stderr: {1}, retcode: {2}'
                    .format(stdout, stderr, retcode))
    return stdout, stderr, retcode


def is_mounted(mount_path):
    stdout, stderr, retcode = run_command('findmnt -n {0}'.format(mount_path), quiet=True)
    if retcode or stdout.split()[0] != mount_path:
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
        fail_and_exit('Failed creating data container',
                      container_name=name,
                      status_code=response.status_code,
                      response=response.text)
    return response.json()


def osmount(fuse_path, dataurl, v3io_mount_path, container='', data_sid=None):
    if not is_mounted(v3io_mount_path):
        run_command('mkdir -p {0}'.format(v3io_mount_path))

        session_arg = '-s {0}'.format(data_sid if data_sid is not None else '')
        print "session:{0}".format(session_arg)
        if container:
            container = '-a ' + container

        command = "nohup {0} -c {1} -m {2} -u on {3} {4} > /dev/null 2>&1 &".\
            format(fuse_path, dataurl, v3io_mount_path, container, session_arg)

        debug_print(command)
        os.system(command)
        for i in [1, 2, 4]:
            if is_mounted(v3io_mount_path):
                break

            if i == 4:
                fail_and_exit('Failed to mount device. Failed to create fuse mount at {0}'.format(v3io_mount_path))

            time.sleep(i)


def load_config():

    try:
        with open(V3IO_CONF_PATH + '/v3io.conf', 'r') as f:
            return json.loads(f.read())
    except Exception as exc:
        fail_and_exit('Failed to open/read v3io conf at {0}'.format(V3IO_CONF_PATH), exc=str(exc))

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

    api_url = clusters['api_url']
    data_url = clusters['data_url']

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
            fail_and_exit('Failed to mount device {0} , Data Container {1} doesn\'t exist'
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


def unmount(mount_path):
    mount_path = os.path.abspath(mount_path)

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


def register_arguments():
    _parser = argparse.ArgumentParser(prog='v3vol', add_help=True)
    sub_parsers = _parser.add_subparsers(dest='action')
    sub_parsers.required = True


    # No additional args needed for those actions
    sub_parsers.add_parser('list', help='List local mounts')
    sub_parsers.add_parser('clear', help='Clear all mounts')
    sub_parsers.add_parser('init', help='No op')
    sub_parsers.add_parser('detach', help='No op')

    # mount
    mount_sub_parser = sub_parsers. \
        add_parser('mount',
                   help='Example: ./v3vol.py mount --mount=dir=/tmp/mymnt '
                        '--json-params=\'{"container":"datalake"}\'')
    mount_sub_parser.add_argument('-md', '--mount-dir', type=str, required=True)
    mount_sub_parser.add_argument('-jp', '--json-params', type=str)

    # unmount
    unmount_sub_parser = sub_parsers. \
        add_parser('unmount', help='Example: ./v3vol.py unmount --mount=dir=/tmp/mymnt')
    unmount_sub_parser.add_argument('-md', '--mount-dir', type=str, required=True)

    return _parser


if __name__ == '__main__':
    parser = register_arguments()
    args = parser.parse_args()

    v3args = load_config()
    DEBUG = v3args['debug']

    debug_print('v3vol arguments: {0}'.format(args))

    if args.action == 'mount':
        mount(args.mount_dir, args.json_params, v3args)
    elif args.action == 'unmount':
        unmount(args.mount_dir)
    elif args.action == 'attach':
        exit_successfully(device='/dev/null')
    elif args.action in ['detach', 'init']:
        exit_successfully()
    elif args.action == 'list':
        os.system('mount | grep v3io')
    elif args.action == 'clear':
        out, err, _ = run_command('mount', quiet=False)

        for l in out.splitlines():
            m = re.match(r'^v3io.*on (.*) type', l, re.M | re.I)
            if m:
                unmount(m.group(1))

