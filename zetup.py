# !! For Iguazio internal usage only !! #

import os
import simplejson
import tempfile

from twisted.internet import defer

import ziggy.utils
import ziggy.fs
import ziggy.docker
import ziggy.tasks
import ziggy.shell


def hook_on_module_load(project):
    base_path = os.path.join(project.config['workspace_path'], project.config['root_projects']['flex-fuse']['path'])

    flex_fuse_config = dict()
    flex_fuse_config['repositories'] = {
        'flex-fuse': project.config['root_projects']['flex-fuse']
    }
    flex_fuse_config['base_path'] = base_path
    flex_fuse_config['flex_fuse_path'] = os.path.join(base_path, 'flex-fuse')
    flex_fuse_config['zetup_path'] = os.path.join(flex_fuse_config['flex_fuse_path'], 'zetup.py')
    flex_fuse_config['zetup_md5'] = ziggy.fs.calculate_file_md5(project.ctx, flex_fuse_config['zetup_path'])

    project.config['flex-fuse'] = flex_fuse_config


@defer.inlineCallbacks
def task_clone(project):
    yield ziggy.tasks.clone(project,
                            repositories=project.config['flex-fuse']['repositories'],
                            base_path=project.config['flex-fuse']['base_path'])


@defer.inlineCallbacks
def task_wait_all_projects_updated(project):
    yield ziggy.tasks.wait_all_projects_updated(project)


@defer.inlineCallbacks
def task_declare_project_updated(project, zetup_path, old_zetup_md5):
    yield ziggy.tasks.declare_project_updated(project, zetup_path, old_zetup_md5)


@defer.inlineCallbacks
def task_update_sources(project):
    yield ziggy.tasks.update_sources(project,
                                     repositories=project.config['flex-fuse']['repositories'],
                                     base_path=project.config['flex-fuse']['base_path'])


@defer.inlineCallbacks
def task_load_snapshot(project, repo_merges=None):
    yield ziggy.tasks.load_snapshot(project,
                                    project_path=project.config['root_projects']['flex-fuse']['path'],
                                    base_path=project.config['flex-fuse']['base_path'],
                                    repo_merges=repo_merges)


@defer.inlineCallbacks
def task_take_snapshot(project, filter=None):
    yield ziggy.tasks.take_snapshot(project,
                                    repositories=project.config['flex-fuse']['repositories'].keys(),
                                    project_path=project.config['root_projects']['flex-fuse']['path'],
                                    base_path=project.config['flex-fuse']['base_path'],
                                    filter=filter)


@defer.inlineCallbacks
def task_verify_zetup_unchanged(project):
    yield ziggy.tasks.wait_workspace_updated(project,
                                             project.config['flex-fuse']['zetup_path'],
                                             project.config['flex-fuse']['zetup_md5'])


@defer.inlineCallbacks
def task_build_images(project, version, mirror=None, nas_deployed_artifacts_path='/mnt/nas'):
    """
    Internal build function
    """

    project.logger.info('Building',
                        version=version,
                        mirror=mirror,
                        nas_deployed_artifacts_path=nas_deployed_artifacts_path)

    cwd = project.config['flex-fuse']['flex_fuse_path']
    cmd = 'make release'

    env = os.environ.copy()
    env['FETCH_METHOD'] = 'download'
    env['MIRROR'] = mirror
    env['IGUAZIO_VERSION'] = version
    env['SRC_BINARY_NAME'] = 'igz-fuse'
    env['DST_BINARY_NAME'] = 'igz-fuse'

    if not mirror:
        env['FETCH_METHOD'] = 'copy'
        env['MIRROR'] = os.path.join(nas_deployed_artifacts_path, 'engine/zeek-packages')

    project.logger.debug('Building a release candidate', cwd=cwd, cmd=cmd, env=env)
    out, _, _ = yield ziggy.shell.run(project.ctx, 'make release', cwd=cwd, env=env)
    project.logger.info('Build images task is done', out=out)


@defer.inlineCallbacks
def task_push_images(project, repository, tag, pushed_images_file_path):
    """
    Internal publish function
    """

    project.logger.info('Pushing images',
                        repository=repository,
                        tag=tag,
                        pushed_images_file_path=pushed_images_file_path)

    cwd = project.config['flex-fuse']['flex_fuse_path']

    repository = repository.replace(tag, '').rstrip('/')

    repository_user = os.path.join('k8s_apps', tag)

    docker_image_name = 'flex-fuse:{0}'.format(tag)
    remote_docker_image_name = '{0}/{1}/{2}'.format(repository, repository_user, docker_image_name)
    cmd = 'docker tag {0} {1}'.format(docker_image_name, remote_docker_image_name)

    # Tag
    project.logger.debug('Tagging docker image before push',
                         cwd=cwd,
                         cmd=cmd,
                         tag=tag,
                         docker_image_name=docker_image_name,
                         remote_docker_image_name=remote_docker_image_name)
    yield ziggy.shell.run(project.ctx, cmd, cwd=cwd)

    # Push
    cmd = 'docker push {0}'.format(remote_docker_image_name)
    project.logger.debug('Pushing docker image to repository',
                         cwd=cwd,
                         cmd=cmd,
                         remote_docker_image_name=remote_docker_image_name)
    yield ziggy.shell.run(project.ctx, cmd, cwd=cwd)

    project.logger.debug('Writing pushed docker image',
                         pushed_images_file_path=pushed_images_file_path,
                         remote_docker_image_name=remote_docker_image_name,
                         docker_image_name=docker_image_name)

    pushed_images = simplejson.dumps([{
        'target_image_name': remote_docker_image_name,
        'image_name': docker_image_name
    }], indent=4)

    ziggy.fs.write_file_contents(project.ctx, pushed_images_file_path, pushed_images)

    project.logger.info('Push images task is done', pushed_images=pushed_images)


@defer.inlineCallbacks
def task_project_build(project, output_dir='flex_fuse_resources', tag='igz',
                       mirror=None, nas_deployed_artifacts_path='/mnt/nas'):
    project.ctx.info('Building', output_dir=output_dir, tag=tag)
    yield ziggy.fs.mkdir(project.ctx, output_dir, force=True)
    tasks_to_run = [
        {
            'name': 'build_images',
            'args': {
                'version': tag,
                'nas_deployed_artifacts_path': nas_deployed_artifacts_path,
                'mirror': mirror,
            },
        },
        {
            'name': 'save_images',
            'args': {
                'output_filepath': os.path.join(output_dir, 'flex-fuse-docker-images.tar.gz'),
                'images': ['flex-fuse:{}'.format(tag)],
            },
        },
    ]

    yield project.task_manager.run_tasks(project, tasks_to_run)
    project.ctx.info('Finished building',
                     output_dir=output_dir,
                     tag=tag,
                     nas_deployed_artifacts_path=nas_deployed_artifacts_path)


@defer.inlineCallbacks
def task_save_images(project, images, output_filepath=None):
    if not output_filepath:
        output_filepath = tempfile.mktemp(suffix='.tar.gz', prefix='flex-fuse-docker-')
        project.logger.debug('no output filepath was given, using a temporary file',
                             output_filepath=output_filepath)

    project.logger.debug('Saving docker images', images=images)
    yield ziggy.docker.save_images(project.ctx, images, output_filepath, compress=True)
    project.logger.debug('Done saving docker images', output_filepath=output_filepath)


@defer.inlineCallbacks
def task_upload(project, upload_manifest_filepath, output_dir):
    project.ctx.info('Uploading',
                     upload_manifest_filepath=upload_manifest_filepath,
                     output_dir=output_dir)

    upload_links_content = ziggy.fs.read_file_contents(project.ctx, upload_manifest_filepath)
    upload_links = simplejson.loads(upload_links_content)

    # we will sync\upload out working dir to each links destination
    for link in upload_links:
        link['src'] = output_dir

    yield ziggy.tasks.upload(project, links=upload_links)

    project.ctx.info('Done uploading',
                     upload_links=upload_links,
                     output_dir=output_dir)


@defer.inlineCallbacks
def task_workflow(project, skipped_tasks=None):
    skipped_tasks = ziggy.utils.as_list(skipped_tasks) or []

    # default workflow
    workflow_tasks = [
        'clone',
        'update_sources',
        'load_snapshot',
        'verify_zetup_unchanged',
        'take_snapshot'
    ]

    # remove tasks we want to skip.
    workflow_tasks = project.task_manager.normalize_task_semantics(workflow_tasks)
    workflow_tasks = [task for task in workflow_tasks if task['name'] not in skipped_tasks]

    yield project.task_manager.run_tasks(project, workflow_tasks)
