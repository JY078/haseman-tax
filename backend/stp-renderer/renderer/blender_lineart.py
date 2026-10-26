#!/usr/bin/env python3
import json
import math
import os
import sys

import bpy
from mathutils import Matrix, Vector


def fail(message: str) -> None:
    print(f"[BlenderLineart] {message}", file=sys.stderr, flush=True)
    sys.exit(1)


def parse_args():
    if "--" not in sys.argv:
        fail("Missing Blender script arguments")

    args = sys.argv[sys.argv.index("--") + 1:]
    values = {}
    i = 0
    while i < len(args):
        key = args[i]
        if not key.startswith("--") or i + 1 >= len(args):
            fail(f"Invalid argument: {key}")
        values[key[2:]] = args[i + 1]
        i += 2

    for key in ("input", "output", "camera", "resolution"):
        if key not in values:
            fail(f"Missing argument: --{key}")

    values["resolution"] = int(values["resolution"])
    values["width"] = int(values.get("width", values["resolution"]))
    values["height"] = int(values.get("height", values["resolution"]))
    values["normalize"] = values.get("normalize", "1") != "0"
    values["mode"] = values.get("mode", "lineart")
    return values


def clear_scene():
    bpy.ops.object.select_all(action="SELECT")
    bpy.ops.object.delete()


def import_model(path: str):
    ext = os.path.splitext(path)[1].lower()
    before = set(bpy.data.objects)

    if ext == ".obj":
        if hasattr(bpy.ops.wm, "obj_import"):
            try:
                bpy.ops.wm.obj_import(filepath=path, forward_axis="Y", up_axis="Z")
            except TypeError:
                bpy.ops.wm.obj_import(filepath=path)
        else:
            try:
                bpy.ops.import_scene.obj(filepath=path, axis_forward="Y", axis_up="Z")
            except TypeError:
                bpy.ops.import_scene.obj(filepath=path)
    elif ext == ".fbx":
        bpy.ops.import_scene.fbx(filepath=path, axis_forward="Y", axis_up="Z", bake_space_transform=False)
    else:
        fail(f"Blender lineart only supports OBJ/FBX for now: {ext}")

    imported = [obj for obj in bpy.data.objects if obj not in before]
    meshes = [obj for obj in imported if obj.type == "MESH"]
    if not meshes:
        meshes = [obj for obj in bpy.context.scene.objects if obj.type == "MESH"]
    if not meshes:
        fail("No mesh objects imported")

    return meshes


def _mesh_bounds(meshes):
    min_v = Vector((math.inf, math.inf, math.inf))
    max_v = Vector((-math.inf, -math.inf, -math.inf))
    for obj in meshes:
        for corner in obj.bound_box:
            world = obj.matrix_world @ Vector(corner)
            min_v.x = min(min_v.x, world.x)
            min_v.y = min(min_v.y, world.y)
            min_v.z = min(min_v.z, world.z)
            max_v.x = max(max_v.x, world.x)
            max_v.y = max(max_v.y, world.y)
            max_v.z = max(max_v.z, world.z)
    return min_v, max_v


def normalize_model(meshes):
    min_v, max_v = _mesh_bounds(meshes)
    center = (min_v + max_v) * 0.5
    source_size = Vector((max_v.x - min_v.x, max_v.y - min_v.y, max_v.z - min_v.z))
    size = max(source_size.x, source_size.y, source_size.z)
    if size <= 0:
        fail("Imported model has empty bounds")

    scale = 2.0 / size
    for obj in meshes:
        world_matrix = obj.matrix_world.copy()
        for vertex in obj.data.vertices:
            world = world_matrix @ vertex.co
            relative = world - center
            vertex.co = relative * scale
        obj.matrix_world = Matrix.Identity(4)
        obj.data.update()

    bpy.context.view_layer.update()


def setup_material(meshes, mode: str):
    materials = []
    for index in range(8):
        mat = bpy.data.materials.new(f"lineart_surface_{index}")
        shade = 0.92 + index * 0.01
        mat.diffuse_color = (shade, shade, shade, 1)
        mat.use_nodes = True
        nodes = mat.node_tree.nodes
        nodes.clear()
        output = nodes.new(type="ShaderNodeOutputMaterial")
        emission = nodes.new(type="ShaderNodeEmission")
        emission.inputs["Color"].default_value = (shade, shade, shade, 1)
        emission.inputs["Strength"].default_value = 1
        mat.node_tree.links.new(emission.outputs[0], output.inputs["Surface"])
        materials.append(mat)

    for obj in meshes:
        original_slots = max(1, len(obj.data.materials))
        obj.data.materials.clear()
        for index in range(original_slots):
            obj.data.materials.append(materials[index % len(materials)])
        for polygon in obj.data.polygons:
            polygon.use_smooth = True
            if original_slots > 0:
                polygon.material_index = min(polygon.material_index, original_slots - 1)
        obj.data.update()


def mark_geometric_edges(meshes, angle_degrees: float = 18.0):
    angle = math.radians(angle_degrees)
    for obj in meshes:
        mesh = obj.data
        mesh.update(calc_edges=True)
        for edge in mesh.edges:
            edge.use_freestyle_mark = False

        edge_faces = {edge.index: [] for edge in mesh.edges}
        edge_key_to_index = {edge.key: edge.index for edge in mesh.edges}
        for poly in mesh.polygons:
            for key in poly.edge_keys:
                idx = edge_key_to_index.get(tuple(sorted(key)))
                if idx is not None:
                    edge_faces[idx].append(poly.index)

        for edge in mesh.edges:
            faces = edge_faces.get(edge.index, [])
            if len(faces) == 1:
                edge.use_freestyle_mark = True
            elif len(faces) >= 2:
                n1 = mesh.polygons[faces[0]].normal
                n2 = mesh.polygons[faces[1]].normal
                if n1.angle(n2, 0.0) >= angle:
                    edge.use_freestyle_mark = True


def camera_from_params(params):
    camera_data = bpy.data.cameras.new("lineart_camera")
    camera = bpy.data.objects.new("lineart_camera", camera_data)
    bpy.context.collection.objects.link(camera)

    pos = Vector(params.get("position", [0, 0, 6]))
    target = Vector(params.get("target", [0, 0, 0]))
    fov = float(params.get("fov", 35))

    camera.location = pos
    if all(key in params for key in ("right", "true_up", "forward")):
        right = Vector(params["right"]).normalized()
        true_up = Vector(params["true_up"]).normalized()
        forward = Vector(params["forward"]).normalized()
        camera.matrix_world = Matrix((
            (right.x, true_up.x, -forward.x, pos.x),
            (right.y, true_up.y, -forward.y, pos.y),
            (right.z, true_up.z, -forward.z, pos.z),
            (0, 0, 0, 1),
        ))
    else:
        direction = target - pos
        if direction.length <= 1e-6:
            direction = Vector((0, 0, -1))
        camera.rotation_euler = direction.to_track_quat("-Z", "Y").to_euler()

    camera.data.type = "PERSP"
    camera.data.angle = math.radians(fov)
    camera.data.sensor_fit = "VERTICAL"
    camera.data.clip_start = 0.001
    camera.data.clip_end = 10000

    bpy.context.scene.camera = camera
    return camera


def fit_camera_to_meshes(camera, meshes, margin: float = 1.25):
    min_v, max_v = _mesh_bounds(meshes)
    corners = [
        Vector((x, y, z))
        for x in (min_v.x, max_v.x)
        for y in (min_v.y, max_v.y)
        for z in (min_v.z, max_v.z)
    ]

    aspect = max(
        bpy.context.scene.render.resolution_x / max(bpy.context.scene.render.resolution_y, 1),
        1e-6,
    )
    tan_y = math.tan(camera.data.angle / 2.0)
    tan_x = tan_y * aspect
    usable = 1.0 / max(margin, 1.0)

    camera_inv = camera.matrix_world.inverted()
    max_backoff = 0.0
    farthest = 0.0
    nearest = math.inf
    for corner in corners:
        local = camera_inv @ corner
        z = -local.z
        if z <= 1e-6:
            max_backoff = max(max_backoff, 1.0 - z)
            z = 1.0
        need_z = max(
            abs(local.x) / max(tan_x * usable, 1e-6),
            abs(local.y) / max(tan_y * usable, 1e-6),
        )
        if need_z > z:
            max_backoff = max(max_backoff, need_z - z)
        farthest = max(farthest, z)
        nearest = min(nearest, z)

    if max_backoff > 0:
        backward = camera.matrix_world.to_quaternion() @ Vector((0, 0, 1))
        camera.location = camera.location + backward * (max_backoff * 1.03)
        bpy.context.view_layer.update()
        farthest += max_backoff
        nearest += max_backoff

    camera.data.clip_start = max(0.001, min(nearest, 1.0) * 0.1)
    camera.data.clip_end = max(1000.0, farthest * 10.0)


def setup_render(width: int, height: int, mode: str):
    scene = bpy.context.scene
    try:
        scene.render.engine = "CYCLES"
        scene.cycles.device = "CPU"
        scene.cycles.samples = 1
    except TypeError:
        scene.render.engine = "BLENDER_EEVEE_NEXT"
    scene.render.film_transparent = False
    scene.render.resolution_x = width
    scene.render.resolution_y = height
    scene.render.resolution_percentage = 100
    scene.render.film_transparent = False
    scene.render.image_settings.color_mode = "RGB"
    scene.render.image_settings.color_depth = "8"
    scene.render.image_settings.file_format = "PNG"
    scene.world = bpy.data.worlds.new("lineart_world") if scene.world is None else scene.world
    scene.world.color = (1, 1, 1)
    scene.world.use_nodes = True
    world_bg = scene.world.node_tree.nodes.get("Background")
    if world_bg:
        world_bg.inputs["Color"].default_value = (1, 1, 1, 1)
        world_bg.inputs["Strength"].default_value = 1
    light_data = bpy.data.lights.new("lineart_light", type="SUN")
    light = bpy.data.objects.new("lineart_light", light_data)
    bpy.context.collection.objects.link(light)
    light.rotation_euler = (math.radians(45), 0, math.radians(45))
    light_data.energy = 2.6

    scene.view_settings.view_transform = "Standard"
    scene.view_settings.look = "None"
    scene.view_settings.exposure = 0
    scene.view_settings.gamma = 1

    scene.render.use_freestyle = mode == "lineart"
    view_layer = scene.view_layers[0]
    view_layer.use_freestyle = mode == "lineart"

    if mode == "depth":
        view_layer.use_pass_z = True
        return

    freestyle = view_layer.freestyle_settings
    lineset = freestyle.linesets[0]
    lineset.select_silhouette = True
    lineset.select_border = True
    lineset.select_crease = False
    lineset.select_edge_mark = False
    lineset.select_contour = True
    lineset.select_material_boundary = False
    lineset.select_suggestive_contour = False
    lineset.select_ridge_valley = False
    lineset.select_external_contour = False
    lineset.select_by_visibility = True
    lineset.visibility = "VISIBLE"

    linestyle = lineset.linestyle
    linestyle.color = (0, 0, 0)
    linestyle.thickness = 1.7

    detail_lineset = freestyle.linesets.new("visible_hard_edge_details")
    detail_lineset.select_silhouette = False
    detail_lineset.select_border = False
    detail_lineset.select_crease = True
    detail_lineset.select_edge_mark = True
    detail_lineset.select_contour = False
    detail_lineset.select_material_boundary = True
    detail_lineset.select_suggestive_contour = False
    detail_lineset.select_ridge_valley = False
    detail_lineset.select_external_contour = False
    detail_lineset.select_by_visibility = True
    detail_lineset.visibility = "VISIBLE"

    detail_linestyle = detail_lineset.linestyle
    detail_linestyle.color = (0, 0, 0)
    detail_linestyle.thickness = 0.9


def render_depth(output_path: str, width: int, height: int):
    scene = bpy.context.scene
    camera = scene.camera
    if camera is None:
        fail("No camera for depth render")

    camera_inv = camera.matrix_world.inverted()
    z_values = []
    depsgraph = bpy.context.evaluated_depsgraph_get()
    for obj in scene.objects:
        if obj.type != "MESH":
            continue
        obj_eval = obj.evaluated_get(depsgraph)
        mesh = obj_eval.to_mesh()
        matrix = obj_eval.matrix_world
        for vertex in mesh.vertices:
            local = camera_inv @ (matrix @ vertex.co)
            z = -local.z
            if z > 1e-6 and math.isfinite(z):
                z_values.append(z)
        obj_eval.to_mesh_clear()

    if z_values:
        near = min(z_values)
        far = max(z_values)
        pad = max((far - near) * 0.05, 0.01)
        near = max(0.001, near - pad)
        far = far + pad
    else:
        near = camera.data.clip_start
        far = camera.data.clip_end

    scene.use_nodes = True
    tree = scene.node_tree
    tree.nodes.clear()
    render_layers = tree.nodes.new(type="CompositorNodeRLayers")
    map_range = tree.nodes.new(type="CompositorNodeMapRange")
    composite = tree.nodes.new(type="CompositorNodeComposite")

    map_range.use_clamp = True
    map_range.inputs["From Min"].default_value = near
    map_range.inputs["From Max"].default_value = far
    map_range.inputs["To Min"].default_value = 1
    map_range.inputs["To Max"].default_value = 0

    depth_output = render_layers.outputs.get("Depth") or render_layers.outputs.get("Z")
    if depth_output is None:
        fail("Render Layers node has no depth output")
    tree.links.new(depth_output, map_range.inputs["Value"])
    tree.links.new(map_range.outputs["Value"], composite.inputs["Image"])

    scene.render.filepath = output_path
    bpy.ops.render.render(write_still=True)


def render(output_path: str):
    bpy.context.scene.render.filepath = output_path
    bpy.context.scene.render.image_settings.file_format = "PNG"
    bpy.ops.render.render(write_still=False)
    bpy.data.images["Render Result"].save_render(filepath=output_path, scene=bpy.context.scene)


def main():
    args = parse_args()
    camera_params = json.loads(args["camera"])

    clear_scene()
    meshes = import_model(args["input"])
    if args["normalize"]:
        normalize_model(meshes)
    setup_material(meshes, args["mode"])
    if args["mode"] == "lineart":
        mark_geometric_edges(meshes)
    setup_render(args["width"], args["height"], args["mode"])
    camera = camera_from_params(camera_params)
    fit_camera_to_meshes(camera, meshes, float(camera_params.get("fit_margin", 1.25)))
    if args["mode"] == "depth":
        render_depth(args["output"], args["width"], args["height"])
    else:
        render(args["output"])


if __name__ == "__main__":
    main()
