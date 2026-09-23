-- +goose Up
ALTER TABLE groups ADD COLUMN scale_images boolean NOT NULL DEFAULT false;
