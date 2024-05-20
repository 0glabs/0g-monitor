import pandas as pd
import re

ip_pattern = "^(http:\/\/|https:\/\/)?\d+.\d+.\d+.\d+$"


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
df.drop_duplicates(
    subset=[
        "Discord id (Fill the text under your profile name instead of the id number)"
    ],
    keep="last",
    inplace=True,
)
df.columns

df = df[
    [
        "Validator address (Fill your 0g account which starts with 0gvaloper)",
        "Validator public RPC Endpoint\r\n\r\n(Add :port if you have custom port. Separate each endpoint by ',' if you have multiple endpoints)",
        "Storage node public IP address\r\n\r\n(Add :port if you have custom port. Separate each endpoint by ',' if you have multiple endpoints)",
        "Storage KV public IP address\r\n\r\n(Add :port if you have custom port. Separate each endpoint by ',' if you have multiple endpoints)",
    ]
]
df.columns = [
    "validator_address",
    "validator_rpc",
    "storage_node_rpc",
    "storage_kv_rpc",
]

df = df[
    df["validator_address"].apply(
        lambda x: type(x) is str and x.startswith("0gvaloper")
    )
]

for col in ["validator_rpc", "storage_node_rpc", "storage_kv_rpc"]:
    df[col] = df[col].apply(lambda x: clean_rpc(x, col))


df.to_csv("user-data/validator_rpcs.csv", index=False)
