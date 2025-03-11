from rest_framework import serializers


class ComputingIDCreateSLZ(serializers.Serializer):
    computing_ids = serializers.ListField(
        label="算力资源标识", child=serializers.CharField(min_length=27), allow_empty=False
    )


class ComputingResourceSLZ(serializers.Serializer):
    computing_id = serializers.CharField(label="计算资源标识", min_length=27)
    power_consumption = serializers.IntegerField(label="功耗")
    cpu_performance = serializers.IntegerField(label="CPU性能")
    cpu_available = serializers.IntegerField(label="CPU可用容量")
    gpu_model = serializers.CharField(label="GPU型号", max_length=255)
    gpu_performance = serializers.IntegerField(label="GPU性能")
    gpu_memory = serializers.IntegerField(label="GPU内存")
    gpu_available = serializers.CharField(label="GPU可用容量", max_length=255)
    network_delay = serializers.IntegerField(label="网络延迟")
    network_performance = serializers.IntegerField(label="网络性能")
    network_isixp = serializers.BooleanField(label="是否为专网")
    network_ips = serializers.CharField(label="IP列表", max_length=255)
    network_available = serializers.CharField(label="网络可用容量", max_length=255)
    network_ips_available = serializers.CharField(label="IP可用资源", max_length=255)
    network_ports = serializers.CharField(label="端口列表", max_length=255)
    price = serializers.IntegerField(label="价格")


class ComputingResourceCreateSLZ(serializers.Serializer):
    computing_resources = serializers.ListField(
        label="算力资源", child=ComputingResourceSLZ(), allow_empty=False
    )


class TaskSLZ(serializers.Serializer):
    source = serializers.CharField(label="请求来源", max_length=255)
    session_identifier = serializers.CharField(label="会话标识", max_length=255)
    data = serializers.DictField(label="请求数据", required=True)
