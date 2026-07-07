import base64
import json

def decode_jwt_payload(token: str) -> dict:
    parts = token.split(".")
    if len(parts) != 3:
        raise ValueError("Invalid JWT format")

    payload = parts[1]

    # Base64URL 补齐 padding
    payload += "=" * (-len(payload) % 4)

    decoded = base64.urlsafe_b64decode(payload)
    return json.loads(decoded)

token = input("Enter JWT token: ")

payload = decode_jwt_payload(token)

print(payload["https://api.openai.com/auth"]["chatgpt_account_id"])


# ca0e29ed-a54c-42d9-a50b-2ba5e065296d

