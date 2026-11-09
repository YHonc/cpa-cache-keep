"""Load the native library and exercise the deployed static/resource split."""
import argparse,base64,ctypes,json,os,pathlib,tempfile
p=argparse.ArgumentParser();p.add_argument('--library',required=True,type=pathlib.Path);a=p.parse_args()
library=a.library.resolve()
class Buffer(ctypes.Structure):_fields_=[('ptr',ctypes.c_void_p),('length',ctypes.c_size_t)]
lib=ctypes.CDLL(str(library))
lib.cliproxyPluginCall.argtypes=[ctypes.c_char_p,ctypes.c_void_p,ctypes.c_size_t,ctypes.POINTER(Buffer)]
lib.cliproxyPluginFree.argtypes=[ctypes.c_void_p,ctypes.c_size_t]
def call(method,payload):
    raw=json.dumps(payload).encode();out=Buffer()
    rc=lib.cliproxyPluginCall(method.encode(),raw,len(raw),ctypes.byref(out))
    try:response=json.loads(ctypes.string_at(out.ptr,out.length))
    finally:lib.cliproxyPluginFree(out.ptr,out.length)
    assert rc==0 and response['ok'],response
    return response['result']
with tempfile.TemporaryDirectory() as tmp:
    previous=os.getcwd();os.chdir(tmp)
    try:
        call('plugin.register',{'config_yaml':base64.b64encode(b'guard_chat: false\n').decode()})
        routes=call('management.register',{})
        assert len(routes['routes'])==2 and all(not r.get('Menu') for r in routes['routes'])
        resource={'Method':'GET','Path':'/v0/resource/plugins/claude-cache-keepalive/status'}
        first=call('management.handle',resource)
        malicious=dict(resource,Query={'format':['json'],'forget':['0123456789abcdef']})
        assert call('management.handle',malicious)['Body']==first['Body']
        state=call('management.handle',{'Method':'GET','Path':'/v0/management/plugins/claude-cache-keepalive/state'})
        assert state['StatusCode']==200 and json.loads(base64.b64decode(state['Body']))['items']==[]
        call('plugin.shutdown',{})
    finally:os.chdir(previous)
print('Native ABI load and static-resource/management routing checks passed')
