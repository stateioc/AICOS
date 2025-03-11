from django.db import models

# Create your models here.


class ComputingID(models.Model):
    """
    算力标识表
    """

    computing_id = models.CharField("算力标识", max_length=255, db_index=True)

    city = models.CharField("用户城市", max_length=255)
    industry = models.CharField("行业", max_length=255)
    organization = models.IntegerField("企业")
    resource_type = models.CharField("资源类型", max_length=255)
    region = models.CharField("城市", max_length=255)
    availability_zone = models.CharField("可用区", max_length=255)
    service_type = models.CharField("服务类型", max_length=255)
    computer_total = models.IntegerField("计算资源")
    storage_total = models.IntegerField("存储资源")
    network_total = models.IntegerField("网络资源")

    class Meta:
        verbose_name = "算力标识表"
        verbose_name_plural = "算力标识表"
        ordering = ["-id"]


class ComputingResourceSKU(models.Model):
    """
    算力资源表
    """

    computing_id = models.ForeignKey(ComputingID, on_delete=models.CASCADE, db_constraint=False)

    power_consumption = models.IntegerField("功耗")
    cpu_performance = models.IntegerField("CPU性能")
    cpu_available = models.IntegerField("CPU可用容量")
    gpu_model = models.CharField("GPU型号", max_length=255)
    gpu_performance = models.IntegerField("GPU性能")
    gpu_memory = models.IntegerField("GPU内存")
    gpu_available = models.CharField("GPU可用容量", max_length=255)
    network_delay = models.IntegerField("网络延迟")
    network_performance = models.IntegerField("网络性能")
    network_isixp = models.BooleanField("是否为专网")
    network_ips = models.CharField("IP列表", max_length=255)
    network_available = models.CharField("网络可用容量", max_length=255)
    network_ips_available = models.CharField("IP可用资源", max_length=255)
    network_ports = models.CharField("端口列表", max_length=255)
    price = models.IntegerField("价格")

    class Meta:
        verbose_name = "算力资源表"
        verbose_name_plural = "算力资源表"
        ordering = ["-id"]


class ComputingResourceRequest(models.Model):
    """
    算力资源请求表
    """

    source = models.CharField("来源", max_length=255)
    session_identifier = models.CharField("会话标识", max_length=255)
    data = models.TextField("请求数据")

    class Meta:
        verbose_name = "算力资源请求表"
        verbose_name_plural = "算力资源请求表"
        ordering = ["-id"]


class TaskTemplate(models.Model):
    """
    任务模版表
    """

    source = models.CharField("来源", max_length=255)
    session_identifier = models.CharField("会话标识", max_length=255)
    data = models.TextField("任务模版数据")

    class Meta:
        verbose_name = "任务模版表"
        verbose_name_plural = "任务模版表"
        ordering = ["-id"]


class TaskResult(models.Model):
    """
    任务结果表
    """

    source = models.CharField("来源", max_length=255)
    session_identifier = models.CharField("会话标识", max_length=255)
    data = models.TextField("任务结果")

    class Meta:
        verbose_name = "任务结果表"
        verbose_name_plural = "任务结果表"
        ordering = ["-id"]
