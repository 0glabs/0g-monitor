import subprocess
import time
import datetime
import os
from dotenv import load_dotenv
from random import randbytes
import pandas as pd

import logging
from logging.handlers import TimedRotatingFileHandler

load_dotenv()

PRIV_KEY = os.getenv("PRIVATE_KEY")


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


data = pd.read_csv('user-data/ips.csv')


block_rpc_endpoint = "http://localhost:8545"
storage_contract_address = "0x1234567890"
file_hashes = {}

while True:
    for _, row in data.iterrows():
        ip = row['ip']
        file_name = f'data/{ip.replace(".", "-")}'
        with open(file_name, "wb") as f:
            random_bytes = randbytes(1048576 * 5)  # 5 MB
            f.write(random_bytes)
        cmd = f"./0g-storage-client/0g-storage-client upload --url {block_rpc_endpoint} --contract {storage_contract_address} --key {PRIV_KEY} --node http://{ip} --file ./{file_name}"
        # execute cmd
        result = subprocess.run(cmd.split(), stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        if result.stderr != "":
            logger.error(f"{ip} failed")
        else:
            logger.info(f"{ip} succeeded")

    time.sleep(600)
