from kfp import dsl
from kfp import kubernetes


@dsl.component
def comp():
    pass

@dsl.pipeline
def my_pipeline():
    task = comp()

    kubernetes.use_secret_as_volume(
        task,
        secret_name='my-hardcoded-secret',
        mount_path='/mnt/my_secret',
    )
