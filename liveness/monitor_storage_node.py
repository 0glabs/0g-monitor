import subprocess
import time
import datetime
import os
import re
from dotenv import load_dotenv
from random import randbytes
import pandas as pd
from threading import Timer
import concurrent.futures

import logging
from logging.handlers import TimedRotatingFileHandler

load_dotenv()

PRIV_KEY = os.getenv("PRIVATE_KEY")

WAIT_SECONDS = 180


def filer(self):
    now = datetime.datetime.now()
    return log_file_name + now.strftime("%Y-%m-%d")


log_file_name = "log/storage_node_log_"
logger = logging.getLogger()
rotating_file_handler = TimedRotatingFileHandler(filename=log_file_name + datetime.datetime.now().strftime("%Y-%m-%d"), when="D", interval=1, backupCount=2)
rotating_file_handler.rotation_filename = filer
formatter = logging.Formatter("%(asctime)s %(name)s:%(levelname)s - %(message)s", "%Y-%m-%d %H:%M:%S")
rotating_file_handler.setFormatter(formatter)
logger.addHandler(rotating_file_handler)
logger.setLevel(logging.INFO)


data = pd.read_csv("user-data/validator_rpcs.csv")
data.fillna("", inplace=True)


kill = lambda process: process.kill()


block_rpc_endpoint = "https://rpc-testnet.0g.ai"
storage_contract_address = "0xb8F03061969da6Ad38f0a4a9f8a86bE71dA3c8E7"
file_hashes = {}


def execute_cmd(r, file_name):
    cmd = f"./0g-storage-client upload --url {block_rpc_endpoint} --contract {storage_contract_address} --key {PRIV_KEY} --node {r} --file ./{file_name}"
    # execute cmd
    result = subprocess.Popen(cmd.split(), stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    my_timer = Timer(WAIT_SECONDS, kill, [result])
    try:
        my_timer.start()
        stdout, _ = result.communicate()
        if "upload took" in stdout:
            is_connected = True
            return True
    finally:
        my_timer.cancel()
    
    return False


def process_item(row):
    user = row["discord_id"]
    validator = row["validator_address"]
    storage_node = row["storage_node_rpc"]

    if len(storage_node) == 0:
        return

    is_connected = False

    file_name = f"data/{re.sub(r'[^A-Za-z0-9-_.~]', '_', storage_node)}.txt"
    with open(file_name, "wb") as f:
        random_bytes = randbytes(1048576 * 1)  # 1 MB
        f.write(random_bytes)

    for r in storage_node.split(","):
        r = r.strip()
        if r.startswith("http"):
            is_connected = execute_cmd(r, file_name)
        else:
            is_connected = execute_cmd(f"http://{r}", file_name)
            
            if not is_connected:
                is_connected = execute_cmd(f"https://{r}", file_name)
        
        if is_connected:
            break

    if is_connected:
        logger.info(f"{storage_node} of {validator} of {user} succeeded")
    else:
        logger.info(f"{storage_node} of {validator} of {user} failed")


while True:
    with concurrent.futures.ThreadPoolExecutor(max_workers=16) as executor:

        futures = [executor.submit(process_item, row) for _, row in data.iterrows()]

        # Wait for all futures to complete
        concurrent.futures.wait(futures)
