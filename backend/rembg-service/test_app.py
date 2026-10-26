import base64
import io
import os
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from fastapi.testclient import TestClient
from PIL import Image

import app


def encoded(image):
    buf = io.BytesIO()
    image.save(buf, "PNG")
    return base64.b64encode(buf.getvalue()).decode()


class BackgroundTests(unittest.TestCase):
    def test_refinement_skips_full_frame_small_and_empty_masks(self):
        for mask in (Image.new("L", (2000, 1500), 255), Image.new("L", (2000, 1500)), Image.new("L", (200, 150), 255)):
            self.assertIsNone(app.refinement_bounds(mask))

    def test_refinement_preserves_disconnected_parts_and_outside_alpha(self):
        mask = Image.new("L", (4000, 3000))
        mask.paste(255, (1200, 1000, 1800, 1600))
        mask.paste(255, (2200, 1100, 2500, 1500))
        mask.putpixel((50, 50), 80)
        bounds = app.refinement_bounds(mask)
        self.assertLess(bounds[0], 1200)
        self.assertGreater(bounds[2], 2500)
        refined = Image.new("L", (bounds[2] - bounds[0], bounds[3] - bounds[1]), 120)
        session = object.__new__(app.EfficientBiRefNetSession)
        with patch.object(app.BiRefNetSessionGeneral, "predict", side_effect=[[mask], [refined]]) as predict:
            result = session.predict(Image.new("RGB", mask.size))[0]
        self.assertEqual(predict.call_count, 2)
        self.assertEqual(result.getpixel((50, 50)), 80)
        self.assertEqual(result.getpixel((1500, 1200)), 120)
        self.assertEqual(result.size, (4000, 3000))

    def setUp(self):
        app.app.state.session = object()
        self.client = TestClient(app.app)

    def test_rejects_invalid_images_and_backgrounds(self):
        self.assertEqual(self.client.post("/remove", json={"image_data": "invalid"}).status_code, 400)
        self.assertEqual(self.client.post("/remove", json={"image_data": "x", "background": "custom"}).status_code, 422)

    def test_offline_model_missing_or_corrupt_fails_before_inference(self):
        with tempfile.TemporaryDirectory() as directory, patch.dict(os.environ, {"U2NET_HOME": directory}):
            with self.assertRaises(FileNotFoundError):
                app.EfficientBiRefNetSession.download_models()
            (Path(directory) / app.MODEL_FILE).write_bytes(b"incomplete upload")
            with self.assertRaises(ValueError):
                app.EfficientBiRefNetSession.download_models()

    def test_health_reports_full_model(self):
        self.assertEqual(self.client.get("/health").json()["model"], "birefnet-general")

    def test_accepts_image_above_previous_pixel_limit(self):
        image = app.decode_image(encoded(Image.new("RGB", (3001, 2000))))
        self.assertEqual(image.size, (3001, 2000))

    def test_preserves_dimensions_and_original_alpha(self):
        source = Image.new("RGBA", (7, 5), (200, 50, 10, 255))
        source.putpixel((0, 0), (0, 0, 0, 0))
        matte = Image.new("RGBA", source.size, (200, 50, 10, 128))
        with patch.object(app, "remove", return_value=matte):
            response = self.client.post("/remove", json={"image_data": encoded(source)})
        self.assertEqual(response.status_code, 200)
        result = Image.open(io.BytesIO(response.content))
        self.assertEqual(result.size, source.size)
        self.assertEqual(result.getpixel((0, 0))[3], 0)
        self.assertEqual(result.getpixel((1, 1))[3], 128)

    def test_solid_backgrounds_and_foreground_preserved(self):
        for background, color in [("white", (255, 255, 255, 255)), ("black", (0, 0, 0, 255))]:
            matte = Image.new("RGBA", (7, 5), (200, 50, 10, 255))
            matte.putpixel((0, 0), (200, 50, 10, 0))
            with patch.object(app, "remove", return_value=matte):
                result = Image.open(io.BytesIO(app.cutout(Image.new("RGBA", (7, 5), (200, 50, 10, 255)), object(), background)))
            self.assertEqual(result.getpixel((0, 0)), color)
            self.assertEqual(result.getpixel((1, 1)), (200, 50, 10, 255))

    def test_product_pixels_unchanged_while_soft_edge_is_decontaminated(self):
        source = Image.new("RGBA", (3, 1), (130, 90, 30, 255))
        matte = Image.new("RGBA", source.size, (100, 80, 20, 255))
        matte.putpixel((1, 0), (100, 80, 20, 120))
        source.putpixel((2, 0), (130, 90, 30, 128))
        with patch.object(app, "remove", return_value=matte):
            result = Image.open(io.BytesIO(app.cutout(source, object(), "transparent")))
        self.assertEqual(result.getpixel((0, 0)), source.getpixel((0, 0)))
        self.assertEqual(result.getpixel((1, 0)), (100, 80, 20, 120))
        self.assertEqual(result.getpixel((2, 0)), source.getpixel((2, 0)))

    def test_busy_and_failure_release_lock(self):
        app.inference_lock.acquire()
        try:
            self.assertEqual(self.client.post("/remove", json={"image_data": "x"}).status_code, 503)
        finally:
            app.inference_lock.release()
        with patch.object(app, "remove", side_effect=RuntimeError("inference failed")):
            self.assertEqual(self.client.post("/remove", json={"image_data": encoded(Image.new("RGB", (2, 2)))}).status_code, 500)
        self.assertFalse(app.inference_lock.locked())


if __name__ == "__main__":
    unittest.main()
