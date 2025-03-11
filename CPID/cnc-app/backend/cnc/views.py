import json

from cnc.models import (
    ComputingID,
    ComputingResourceRequest,
    ComputingResourceSKU,
    TaskResult,
    TaskTemplate,
)
from rest_framework.response import Response
from rest_framework.viewsets import GenericViewSet

from .authentications import TokenAuthentication
from .renderers import ResponseRenderer
from .serializers import ComputingIDCreateSLZ, ComputingResourceCreateSLZ, TaskSLZ
from .utils import parse_computing_id


class ComputingIDViewSet(GenericViewSet):
    """创建算力标识"""

    authentication_classes = [TokenAuthentication]
    renderer_classes = [ResponseRenderer]

    serializer_class = ComputingIDCreateSLZ

    def create(self, request, *args, **kwargs):
        serializer = self.get_serializer(data=request.data)
        serializer.is_valid(raise_exception=True)

        data = serializer.validated_data

        computing_ids = []
        for _id in set(data["computing_ids"]):
            if ComputingID.objects.filter(computing_id=_id).exits():
                continue

            tuple_id = parse_computing_id(_id)
            computing_id = ComputingID(computing_id=_id, **tuple_id._asdict())
            computing_ids.append(computing_id)

        if computing_ids:
            ComputingID.objects.bulk_create(computing_ids, batch_size=100)

        return Response({})


class ComputingResourceSKUViewSet(GenericViewSet):
    """创建算力资源"""

    authentication_classes = [TokenAuthentication]
    renderer_classes = [ResponseRenderer]

    serializer_class = ComputingResourceCreateSLZ

    def create(self, request, *args, **kwargs):
        serializer = self.get_serializer(data=request.data)
        serializer.is_valid(raise_exception=True)

        data = serializer.validated_data

        computing_resources = []
        for data in data["computing_resources"]:
            computing_id = ComputingID.objects.filter(
                computing_id=data["computing_id"]
            ).first()
            if not computing_id:
                raise ValueError(f"computing_id {data['computing_id']} not found")

            computing_resource = ComputingResourceSKU(
                computing_id=computing_id,
                **{k: v for k, v in data.items() if k != "computing_id"},
            )
            computing_resources.append(computing_resource)

        ComputingResourceSKU.objects.bulk_create(computing_resources, batch_size=100)

        return Response({})


class QueryComputingResourceViewSet(GenericViewSet):
    """查询算力资源"""

    authentication_classes = [TokenAuthentication]
    renderer_classes = [ResponseRenderer]

    serializer_class = TaskSLZ

    def create(self, request, *args, **kwargs):
        serializer = self.get_serializer(data=request.data)
        serializer.is_valid(raise_exception=True)

        data = serializer.validated_data

        m = ComputingResourceRequest(
            source=data["source"],
            session_identifier=data["session_identifier"],
            data=json.dumps(data["data"]),
        )
        m.save()

        return Response({})


class QueryComputingResourceByTaskTemplateViewSet(GenericViewSet):
    """通过任务模板查询算力资源"""

    authentication_classes = [TokenAuthentication]
    renderer_classes = [ResponseRenderer]

    serializer_class = TaskSLZ

    def create(self, request, *args, **kwargs):
        serializer = self.get_serializer(data=request.data)
        serializer.is_valid(raise_exception=True)

        data = serializer.validated_data

        m = TaskTemplate(
            source=data["source"],
            session_identifier=data["session_identifier"],
            data=json.dumps(data["data"]),
        )
        m.save()

        return Response({})


class TaskPathViewSet(GenericViewSet):
    """任务路径反馈"""

    authentication_classes = [TokenAuthentication]
    renderer_classes = [ResponseRenderer]

    serializer_class = TaskSLZ

    def create(self, request, *args, **kwargs):
        serializer = self.get_serializer(data=request.data)
        serializer.is_valid(raise_exception=True)

        data = serializer.validated_data

        m = TaskResult(
            source=data["source"],
            session_identifier=data["session_identifier"],
            data=json.dumps(data["data"]),
        )
        m.save()

        return Response({})
