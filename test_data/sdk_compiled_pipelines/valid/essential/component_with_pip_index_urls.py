# Copyright 2022 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import os

from kfp import compiler, kubernetes
from kfp import dsl
from kfp.dsl import component, PipelineTask

PACKAGES_TO_INSTALL = ['yapf']
if 'KFP_PIPELINE_SPEC_PACKAGE_PATH' in os.environ:
    PACKAGES_TO_INSTALL.append(os.environ['KFP_PIPELINE_SPEC_PACKAGE_PATH'])

def add_pip_index_configuration(task: PipelineTask):
    kubernetes.use_config_map_as_env(
        task,
        config_map_name="ds-pipeline-custom-env-vars",
        config_map_key_to_env={"pip_index_url": "PIP_INDEX_URL", "pip_trusted_host": "PIP_TRUSTED_HOST"},
    )


@component(
    packages_to_install=PACKAGES_TO_INSTALL)
def component_op():
    import yapf
    print(dir(yapf))


@dsl.pipeline(name='v2-component-pip-index-urls')
def pipeline():
    task = component_op()
    add_pip_index_configuration(task)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=__file__.replace('.py', '.yaml'))
