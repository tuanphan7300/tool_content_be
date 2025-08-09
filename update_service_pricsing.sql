ALTER TABLE `tool`.`service_pricings`
    CHANGE COLUMN `price_per_unit` `price_per_unit` DECIMAL(15,10) NOT NULL ;

ALTER TABLE `tool`.`credit_transactions`
    CHANGE COLUMN `amount` `amount` DECIMAL(15,10) NOT NULL ,
    CHANGE COLUMN `units_used` `units_used` DECIMAL(15,10) NULL DEFAULT NULL ;


INSERT INTO service_pricings (service_name, model_api_name, pricing_type, price_per_unit, currency, description, is_active)
VALUES ('gemini-2.0-flash_input','gemini-2.0-flash','per_token',0.00000015,'USD','Gemini 2.0 Flash INPUT - $0.15 per 1M tokens (~$0.0375/1M chars)',1)
    ON DUPLICATE KEY UPDATE price_per_unit=VALUES(price_per_unit), model_api_name=VALUES(model_api_name), description=VALUES(description), is_active=VALUES(is_active);

INSERT INTO service_pricings (service_name, model_api_name, pricing_type, price_per_unit, currency, description, is_active)
VALUES ('gemini-2.0-flash_output','gemini-2.0-flash','per_token',0.00000060,'USD','Gemini 2.0 Flash OUTPUT - $0.60 per 1M tokens (~$0.15/1M chars)',1)
    ON DUPLICATE KEY UPDATE price_per_unit=VALUES(price_per_unit), model_api_name=VALUES(model_api_name), description=VALUES(description), is_active=VALUES(is_active);

UPDATE `tool`.`service_pricings` SET `service_name` = 'gemini-1.5-flash_input', `price_per_unit` = '0.000000075', `description` = 'Gemini 1.5 Flash OUTPUT - $0.075 per 1M tokens (~$0.15/1M chars)' WHERE (`id` = '2');
INSERT INTO `tool`.`service_pricings` (`id`, `service_name`, `model_api_name`, `pricing_type`, `price_per_unit`, `currency`, `description`, `is_active`, `created_at`, `updated_at`) VALUES ('19', 'gemini-1.5-flash_output', 'gemini-1.5-flash', 'per_token', '0.0000003', 'USD', 'Gemini 1.5 Flash OUTPUT - $0.30 per 1M tokens (~$0.15/1M chars)', '1', '2025-07-08 10:26:07', '2025-07-12 23:23:03');

UPDATE `tool`.`service_pricings` SET `price_per_unit` = '0.0000075' WHERE (`id` = '2');
UPDATE `tool`.`service_pricings` SET `price_per_unit` = '0.00003' WHERE (`id` = '19');
UPDATE `tool`.`service_pricings` SET `price_per_unit` = '0.0000105' WHERE (`id` = '15');
UPDATE `tool`.`service_pricings` SET `price_per_unit` = '0.0001' WHERE (`id` = '18');
