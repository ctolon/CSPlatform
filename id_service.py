from fastapi import FastAPI, HTTPException, Header
import pwd
# import grp
import os
from pathlib import Path
import ssl
from dotenv import load_dotenv

load_dotenv()

app = FastAPI(
    docs_url=None,
    redoc_url=None,     
    openapi_url=None    
)

API_KEY = os.getenv("API_KEY", None)

cert = str(Path(__file__).parent.resolve() / "server_certificate.pem")
key = str(Path(__file__).parent.resolve() / "private.key")

ssl._create_default_https_context = ssl._create_unverified_context

@app.get("/user-info/{username}")
def get_user_info(username: str, x_api_key: str = Header(None)):
    
    if x_api_key != API_KEY:
            raise HTTPException(status_code=401, detail="Invalid or missing API Key")
    
    try:
        user = pwd.getpwnam(username)
        uid = user.pw_uid
        gid = user.pw_gid
        
        return {
            "username": username,
            "uid": uid,
            "gid": gid,
        }
    except KeyError:
        raise HTTPException(status_code=404, detail=f"User '{username}' not found")
    
if __name__ == "__main__":
    import uvicorn
    import argparse
    
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", "-ho", type=str, default="0.0.0.0")
    parser.add_argument("--port", "-p", type=int, default=4044)
    parser.add_argument("--with-ssl", "-s", action="store_true")
    parser.add_argument("--cert-path", "-c", type=str, default=cert)
    parser.add_argument("--key-path", "-k", type=str, default=key)
    parser.add_argument("--log-level", "-l", type=str, default="info")
    args = parser.parse_args()
    host = args.host
    port = args.port
    with_ssl = args.with_ssl
    cert_path = args.cert_path
    key_path = args.key_path
    log_level = args.log_level
    if with_ssl:
        uvicorn.run(app, host=host, port=port, log_level=log_level, ssl_keyfile=key_path, ssl_certfile=cert_path)
    else:
        uvicorn.run(app, host=host, port=port, log_level=log_level)