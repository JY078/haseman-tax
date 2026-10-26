CREATE INDEX IF NOT EXISTS idx_scene_style_references_scene_tag_id
    ON scene_style_references (scene_id, tag_id, id);

CREATE INDEX IF NOT EXISTS idx_scene_style_references_tag_scene_id
    ON scene_style_references (tag_id, scene_id, id);
