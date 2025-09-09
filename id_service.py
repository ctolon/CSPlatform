from fastapi import FastAPI, HTTPException
import pwd
# import grp

app = FastAPI(
    docs_url=None,
    redoc_url=None,     
    openapi_url=None    
)

@app.get("/user-info/{username}")
def get_user_info(username: str):
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
    uvicorn.run(app, host="0.0.0.0", port=4044)