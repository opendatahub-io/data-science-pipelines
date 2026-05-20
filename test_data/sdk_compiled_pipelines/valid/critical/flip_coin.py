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
from kfp import compiler, dsl, kubernetes
from kfp.dsl import PipelineTask


def add_pip_index_configuration(task: PipelineTask):
    kubernetes.use_config_map_as_env(
        task,
        config_map_name="ds-pipeline-custom-env-vars",
        config_map_key_to_env={"pip_index_url": "PIP_INDEX_URL", "pip_trusted_host": "PIP_TRUSTED_HOST"},
    )

@dsl.component()
def random_num(low: int, high: int) -> int:
    """Generate a random number between low and high."""
    import random  # noqa: PLC0415

    result = random.randint(low, high)
    print(result)
    return result


@dsl.component()
def flip_coin() -> str:
    """Flip a coin and output heads or tails randomly."""
    import random  # noqa: PLC0415

    result = "heads" if random.randint(0, 1) == 0 else "tails"
    print(result)
    return result


@dsl.component()
def print_msg(msg: str):
    """Print a message."""
    print(msg)

@dsl.pipeline(
    name="conditional-execution-pipeline",
    description="Shows how to use dsl.If().",
)
def flipcoin_pipeline():
    flip = flip_coin().set_caching_options(False)
    add_pip_index_configuration(flip)
    with dsl.If(flip.output == "heads"):
        random_num_head = random_num(low=0, high=9).set_caching_options(False)
        add_pip_index_configuration(random_num_head)
        with dsl.If(random_num_head.output > 5):
            print_head_gt = print_msg(msg="heads and %s > 5!" % random_num_head.output).set_caching_options(False)
            add_pip_index_configuration(print_head_gt)
        with dsl.If(random_num_head.output <= 5):
            print_head_le = print_msg(msg="heads and %s <= 5!" % random_num_head.output).set_caching_options(False)
            add_pip_index_configuration(print_head_le)

    with dsl.If(flip.output == "tails"):
        random_num_tail = random_num(low=10, high=19).set_caching_options(False)
        add_pip_index_configuration(random_num_tail)
        with dsl.If(random_num_tail.output > 15):
            print_tail_gt = print_msg(msg="tails and %s > 15!" % random_num_tail.output).set_caching_options(False)
            add_pip_index_configuration(print_tail_gt)
        with dsl.If(random_num_tail.output <= 15):
            print_tail_le = print_msg(msg="tails and %s <= 15!" % random_num_tail.output).set_caching_options(False)
            add_pip_index_configuration(print_tail_le)


if __name__ == "__main__":
    compiler.Compiler().compile(flipcoin_pipeline, package_path=__file__.replace(".py", ".yaml"))
