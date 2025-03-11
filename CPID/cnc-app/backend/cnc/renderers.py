from rest_framework.renderers import JSONRenderer


class ResponseRenderer(JSONRenderer):
    """
    采用统一的结构封装返回内容
    """

    SUCCESS_CODE = 0
    SUCCESS_MESSAGE = "OK"

    def render(self, data, accepted_media_type=None, renderer_context=None):
        if not isinstance(data, dict) or "result" not in data or "code" not in data:
            data = {
                "data": data,
                "result": True,
                "code": self.SUCCESS_CODE,
                "message": self.SUCCESS_MESSAGE,
            }

        if renderer_context and "permissions" in renderer_context:
            data["permissions"] = renderer_context["permissions"]

        if renderer_context and "message" in renderer_context:
            data["message"] = renderer_context["message"]

        response = super().render(data, accepted_media_type, renderer_context)
        return response
