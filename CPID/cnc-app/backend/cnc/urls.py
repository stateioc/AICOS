from django.urls import path

from . import views

urlpatterns = [
    path(
        "computing_ids/",
        views.ComputingIDViewSet.as_view({"post": "create"}),
        name="computing_ids",
    ),
    path(
        "computing_resources/",
        views.ComputingResourceSKUViewSet.as_view({"post": "create"}),
        name="computing_resources",
    ),
    path(
        "query_computing_resources/",
        views.QueryComputingResourceViewSet.as_view({"post": "create"}),
        name="query_computing_resources",
    ),
    path(
        "query_computing_resources_by_task_template/",
        views.QueryComputingResourceByTaskTemplateViewSet.as_view({"post": "create"}),
        name="query_computing_resources_by_task_template",
    ),
    path(
        "task_path/",
        views.TaskPathViewSet.as_view({"post": "create"}),
        name="task_path",
    ),
]
