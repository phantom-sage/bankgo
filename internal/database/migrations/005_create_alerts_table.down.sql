-- Drop alerts table and related objects
DROP TRIGGER IF EXISTS trigger_alerts_updated_at ON alerts;
DROP FUNCTION IF EXISTS update_alerts_updated_at();
DROP TABLE IF EXISTS alerts;