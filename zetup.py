# !! For Iguazio internal usage only !! #

import os
import simplejson

from twisted.internet import defer

import ziggy.utils
import ziggy.fs
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
def task_build_images(project, version, mirror):
    """
    Internal build function
    """

    project.logger.debug('Building', version=version, mirror=mirror)

    cwd = project.config['flex-fuse']['flex_fuse_path']
    cmd = 'make release MIRROR={0} IGUAZIO_VERSION={1}'.format(mirror, version)

    project.logger.debug('Building a release candidate', cwd=cwd, cmd=cmd)

    out, _, _ = yield ziggy.shell.run(project.ctx, cmd, cwd=cwd)

    project.logger.debug('Build images task is done', out=out)


@defer.inlineCallbacks
def task_push_images(project, repository, tag, pushed_images_file_path):
    """
    Internal publish function
    """

    project.logger.debug('Pushing images',
                         repository=repository,
                         tag=tag,
                         pushed_images_file_path=pushed_images_file_path)

    cwd = project.config['flex-fuse']['flex_fuse_path']

    version_path = os.path.join(cwd, 'VERSION')
    project.logger.debug('Collecting output version', version_path=version_path)
    image_tag = ziggy.fs.read_file_contents(project.ctx, version_path).strip()

    docker_image_name = 'flex-fuse:{0}'.format(image_tag)
    remote_docker_image_name = '{0}/{1}'.format(repository, docker_image_name)
    cmd = 'docker tag {0} {1}'.format(docker_image_name, remote_docker_image_name)

    # Tag
    project.logger.debug('Tagging docker image before push',
                         cwd=cwd,
                         cmd=cmd,
                         image_tag=image_tag,
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

    pushed_image = {
        'target_image_name': remote_docker_image_name,
        'image_name': docker_image_name
    }
    ziggy.fs.write_file_contents(project.ctx, pushed_images_file_path, simplejson.dumps([pushed_image]))

    project.logger.debug('Push images task is done')


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
