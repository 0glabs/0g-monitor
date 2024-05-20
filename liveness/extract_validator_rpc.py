import pandas as pd
import re


def clean_rpc(v: str, t: str):
    port = 8545
    if t == "storage_node_rpc":
        port = 5678
    elif t == "storage_kv_rpc":
        port = 6789
    if type(v) is not str or len(v) == 0:
        return ""
    rpcs = v.split(",")
    res = []
    for rpc in rpcs:
        match = re.match(ip_pattern, rpc)
        if match:
            res.append(f"{rpc}:{port}")
        else:
            res.append(rpc)
    return ",".join(res)


df = pd.read_csv("contributors.csv", parse_dates=["Timestamp"])
df.sort_values(by="Timestamp", inplace=True, ascending=True)
df.drop_duplicates(subset=["Discord id"], keep="last", inplace=True)

df = df[
    [
        "Validator address \n\n(Fill the evmosvaloper_ if you have past activities, otherwise fill 0gvaloper_)",
        "Validator public RPC Endpoint (if applicable) (ip:port)",
        "Storage node public IP address (if applicable)   (ip:port)",
        "Storage KV public IP address (if applicable)  (ip:port)",
    ]
]
df.columns = ["validator_address", "validator_rpc", "storage_node_rpc", "storage_kv_rpc"]

df = df[df["validator_address"].apply(lambda x: type(x) is str and x.startswith("0gvaloper"))]

ip_pattern = "^(http:\/\/|https:\/\/)?\d+.\d+.\d+.\d+$"

for col in ["validator_rpc", "storage_node_rpc", "storage_kv_rpc"]:
    df[col] = df[col].apply(lambda x: clean_rpc(x, col))


df.to_csv("liveness/user-data/validator_rpcs.csv", index=False)
