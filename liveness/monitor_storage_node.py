import subprocess
import time
import datetime
import os
from dotenv import load_dotenv
from random import randbytes
import pandas as pd
from threading import Timer

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


data = pd.read_csv('user-data/validator_rpcs.csv')
data.fillna('', inplace=True)


kill = lambda process: process.kill()


block_rpc_endpoint = "https://rpc-testnet.0g.ai"
storage_contract_address = "0x2b8bC93071A6f8740867A7544Ad6653AdEB7D919"
file_hashes = {}

while True:
    for _, row in data.iterrows():
        validator = row['validator_address']
        storage_node = row['storage_node_rpc']
        
        if len(storage_node) == 0:
            continue
        
        is_connected = False
        
        file_name = f'data/{validator.replace(".", "-")}'
        with open(file_name, "wb") as f:
            random_bytes = randbytes(1048576 * 1)  # 1 MB
            f.write(random_bytes)
            
        for r in storage_node.split(','):
            cmd = f"./0g-storage-client upload --url {block_rpc_endpoint} --contract {storage_contract_address} --key {PRIV_KEY} --node http://{r} --file ./{file_name}"
            # execute cmd
            result = subprocess.Popen(cmd.split(), stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
            my_timer = Timer(180, kill, [result])
            try:
                my_timer.start()
                stdout, _ = result.communicate()
                if "upload took" in stdout:
                    is_connected = True
                    break
            finally:
                my_timer.cancel()
        
        if is_connected:
            logger.info(f"{storage_node} of {validator} succeeded")
        else:
            logger.info(f"{storage_node} of {validator} failed")

    time.sleep(600)
