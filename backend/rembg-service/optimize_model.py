"""Build the offline FP32 graph: python optimize_model.py SOURCE.onnx OUTPUT.onnx.

Build-only dependency: onnx==1.23.0. No retraining or quantization.
Per-tap GridSample approach also used by CoderViking/birefnet-lite-onnx:
https://huggingface.co/CoderViking/birefnet-lite-onnx
This converter preserves the pinned FULL rembg model, including its backbone.
"""

import argparse
import hashlib
from pathlib import Path
import onnx
import numpy as np
from onnx import helper as h, numpy_helper as nh

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("source", type=Path)
parser.add_argument("output", type=Path)
args = parser.parse_args()
source = args.source
with source.open("rb") as original:
    if (
        hashlib.file_digest(original, "md5").hexdigest()
        != "7a35a0141cbbc80de11d9c9a28f52697"
    ):
        raise ValueError(
            "Expected the pinned rembg BiRefNet-general-epoch_244.onnx export"
        )
m = onnx.load(source)
initializers = {t.name: t for t in m.graph.initializer}
producers = {o: n for n in m.graph.node for o in n.output}


def array(name):
    if name in initializers:
        return nh.to_array(initializers[name])
    n = producers[name]
    if n.op_type == "Identity":
        return array(n.input[0])
    assert n.op_type == "Constant"
    return nh.to_array(n.attribute[0].t)


new_nodes = []
new_weights = []
count = 0
prefixes = [
    n.name.removesuffix("offset_conv/Conv")
    for n in m.graph.node
    if "/atrous_conv/offset_conv/Conv" in n.name
]
replacements = {}
skip = set()
for prefix in prefixes:
    ns = [n for n in m.graph.node if n.name.startswith(prefix)]
    byname = {n.name: n for n in ns}
    offset = byname[prefix + "offset_conv/Conv"]
    mod = byname[prefix + "modulator_conv/Conv"]
    conv = byname[prefix + "Conv"]
    weights = array(conv.input[1])
    attrs = {a.name: h.get_attribute_value(a) for a in offset.attribute}
    kh, kw = attrs["kernel_shape"]
    pad = attrs["pads"][0]
    assert attrs["strides"] == [1, 1] and attrs["dilations"] == [1, 1]
    dims = array(byname[prefix + "Constant_1"].output[0])
    H, W = map(int, dims[-2:])
    nodes = ns[:5]
    assert nodes[-1].op_type == "Mul"
    counter = 0

    def const(value, dtype=np.float32):
        global counter
        name = prefix + "grid_const_" + str(counter)
        counter += 1
        new_weights.append(nh.from_array(np.asarray(value, dtype=dtype), name))
        return name

    def op(kind, *inputs, **attrs):
        global counter
        name = prefix + "grid_" + str(counter)
        counter += 1
        nodes.append(h.make_node(kind, list(inputs), [name], name=name, **attrs))
        return name

    # Keeping the original offset and modulation convolutions preserves weights.
    ys, xs = np.meshgrid(
        np.arange(H, dtype=np.float32), np.arange(W, dtype=np.float32), indexing="ij"
    )
    gx = const((2 * xs + 1) / W - 1)
    gy = const((2 * ys + 1) / H - 1)
    sx = const(2 / W)
    sy = const(2 / H)
    axes = const([3], np.int64)
    accum = None
    for k in range(kh * kw):
        iy, ix = divmod(k, kw)
        dy = op("Gather", offset.output[0], const(2 * k, np.int64), axis=1)
        dx = op("Gather", offset.output[0], const(2 * k + 1, np.int64), axis=1)
        py = op("Add", gy, op("Mul", op("Add", dy, const(iy - pad)), sy))
        px = op("Add", gx, op("Mul", op("Add", dx, const(ix - pad)), sx))
        grid = op(
            "Concat", op("Unsqueeze", px, axes), op("Unsqueeze", py, axes), axis=3
        )
        sampled = op(
            "GridSample",
            offset.input[0],
            grid,
            align_corners=0,
            mode="bilinear",
            padding_mode="zeros",
        )
        modulation = op("Gather", ns[4].output[0], const([k], np.int64), axis=1)
        scaled = op("Mul", sampled, modulation)
        weight = const(weights[:, :, iy, ix, None, None])
        contribution = op("Conv", scaled, weight, kernel_shape=[1, 1])
        accum = contribution if accum is None else op("Add", accum, contribution)
    # Original final Add contains the deformable convolution bias.
    final = ns[-1]
    assert final.op_type == "Add"
    bias = const(array(final.input[1]))
    nodes.append(h.make_node("Add", [accum, bias], list(final.output), name=final.name))
    outputs = {o for n in ns for o in n.output}
    used = {i for n in m.graph.node if n not in ns for i in n.input}
    assert outputs & used == set(final.output)
    replacements[ns[0].name] = nodes
    skip.update(n.name for n in ns)
    count += 1
for n in m.graph.node:
    if n.name in replacements:
        new_nodes.extend(replacements[n.name])
    elif n.name not in skip:
        new_nodes.append(n)
del m.graph.node[:]
m.graph.node.extend(new_nodes)
used = {i for n in new_nodes for i in n.input}
keep = [t for t in m.graph.initializer if t.name in used] + new_weights
del m.graph.initializer[:]
m.graph.initializer.extend(keep)
# The source is opset 17, which already supports GridSample.
onnx.checker.check_model(m)
assert count == 20
out = args.output
onnx.save(m, out)
print(
    "Converted", count, "deformable convolutions:", Path(out).stat().st_size, flush=True
)
