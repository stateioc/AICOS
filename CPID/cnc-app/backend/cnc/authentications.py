from rest_framework.authentication import BasicAuthentication
from django.contrib.auth import get_user_model


token_set = {"bbc73c45893e41e8932ecfc267e20ce0", "046d27ea2aff4ec2a9a99dda6687349d"}


class TokenAuthentication(BasicAuthentication):
    def authenticate(self, request):
        token = request.headers.get("token")
        if token not in token_set:
            return None
        user = get_user_model().objects.all().first()
        return (user, None)
