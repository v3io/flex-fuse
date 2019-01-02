import groovy.json.JsonOutput


ZARCHO_GIT = "git@github.com:${params.zarcho_user}/zarcho.git"
GITHUB_AUTH = 'github_auth_id'


@Library('pipelinex@development')
import com.iguazio.pipelinex.DockerRepo


library identifier: "zarcho@${params.zarcho_branch}", retriever: modernSCM(
    [$class: 'GitSCMSource',
     credentialsId: GITHUB_AUTH,
     remote: ZARCHO_GIT]) _


builder.set_job_properties([
    string(defaultValue: 'REPLACE_ME', description: '', name: 'build_version'),

    string(defaultValue: 'development', description: '', name: 'flex_fuse_branch'),
    string(defaultValue: 'v3io', description: '', name: 'flex_fuse_user'),

    string(defaultValue: 'next', description: '', name: 'zarcho_branch'),
    string(defaultValue: 'iguazio', description: '', name: 'zarcho_user'),

    string(defaultValue: 'short', description: '', name: 'workflow'),

    booleanParam(defaultValue: false, description: '', name: 'publish_to_public_registries'),
])


common.notify_slack {
    common.set_current_display_name(params.build_version)

    stage('git clone') {
        nodes.any_builder_node {
            builder.clone_zarcho(params.zarcho_user, params.zarcho_branch)
        }
    }

    def snapshot = ['k8s-flex-fuse':
                        ['flex-fuse':
                            ['branch': params.flex_fuse_branch,
                             'git_url': "git@github.com:${params.flex_fuse_user}/flex-fuse.git",
                            ]
                        ]
                    ]

    k8s.build_flex_fuse(params.build_version, JsonOutput.toJson(snapshot), params.workflow,
                        params.publish_to_public_registries)
}
