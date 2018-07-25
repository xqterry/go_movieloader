from PIL import Image
import numpy as np


def get_image(buffer, frame_size, pos = 0, fw= 448, fh=448, ps=3):
    dd = np.frombuffer(buffer, dtype=np.uint8, count=frame_size, offset=pos)

    dd = dd.reshape(fh, fw, ps)
    img = Image.fromarray(dd, 'RGB')

    return img

import argparse

parser = argparse.ArgumentParser(description='Train')
parser.add_argument("--input", type=str, default='', help="rgb file")
parser.add_argument("--fw", type=int, default=896, help="w")
parser.add_argument("--fh", type=int, default=448, help="h")
parser.add_argument("--ps", type=int, default=3, help="s")

opt = parser.parse_args()

import os
from os.path import basename

sz = os.path.getsize(opt.input)
with open(opt.input, "rb") as f:
    buf = f.read(sz)
    print("file size ", sz)
    img = get_image(buf, opt.fh * opt.fw * opt.ps, fw=opt.fw, fh=opt.fh)

    img.save(basename(opt.input) + ".png")
