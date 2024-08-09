from kfp import dsl
from kfp import kubernetes


@dsl.component
def comp():
    pass

@dsl.pipeline
def my_pipeline(my_secret: str):
    task = comp()
    task.set_caching_options(False)

    kubernetes.use_secret_as_volume(
        task,
        secret_name=my_secret,
        mount_path='/mnt/my_secret',
    )
