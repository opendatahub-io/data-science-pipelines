from kfp import dsl
from kfp import kubernetes


@dsl.component
def comp():
    pass

@dsl.pipeline
def my_pipeline():
    task = comp()

    my_secret = 'my-hardcoded-secret2'

    kubernetes.use_secret_as_volume(
        task,
        secret_name=my_secret,
        mount_path='/mnt/my_secret',
    )
