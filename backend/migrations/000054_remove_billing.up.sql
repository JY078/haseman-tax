DROP TABLE IF EXISTS recharge_orders;
DROP TABLE IF EXISTS point_logs;

ALTER TABLE users DROP COLUMN IF EXISTS points;
ALTER TABLE ai_models DROP COLUMN IF EXISTS billing_mode;
ALTER TABLE ai_models DROP COLUMN IF EXISTS price_per_use;
ALTER TABLE ai_models DROP COLUMN IF EXISTS price_input;
ALTER TABLE ai_models DROP COLUMN IF EXISTS price_output;
ALTER TABLE usage_logs DROP COLUMN IF EXISTS cost;
ALTER TABLE IF EXISTS white_bg_tasks DROP COLUMN IF EXISTS points_cost;
ALTER TABLE IF EXISTS multi_angle_tasks DROP COLUMN IF EXISTS points_cost;

DELETE FROM settings
WHERE key IN (
    'payment_enabled',
    'payment_provider',
    'recharge_plans',
    'alipay_app_id',
    'alipay_private_key',
    'alipay_public_key',
    'alipay_notify_url',
    'alipay_sandbox',
    'ysm_app_id',
    'ysm_secret',
    'ysm_pay_type',
    'ysm_notify_url'
);

DELETE FROM menus
WHERE path IN (
    '/points',
    '/recharges',
    '/recharge-plans',
    '/image-slice',
    '/vectorize',
    '/resize',
    '/layer-editor'
);
