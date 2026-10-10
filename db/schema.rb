# This file is auto-generated from the current state of the database. Instead
# of editing this file, please use the migrations feature of Active Record to
# incrementally modify your database, and then regenerate this schema definition.
#
# This file is the source Rails uses to define your schema when running `bin/rails
# db:schema:load`. When creating a new database, `bin/rails db:schema:load` tends to
# be faster and is potentially less error prone than running all of your
# migrations from scratch. Old migrations may fail to apply correctly if those
# migrations use external dependencies or application code.
#
# It's strongly recommended that you check this file into your version control system.

ActiveRecord::Schema[8.1].define(version: 2026_10_10_034152) do
  create_table "invoices", force: :cascade do |t|
    t.integer "user_id", null: false
    t.string "stripe_invoice_id", null: false
    t.integer "amount_cents", default: 0, null: false
    t.string "currency", default: "usd", null: false
    t.string "status", default: "open", null: false
    t.string "invoice_pdf_url", default: "", null: false
    t.datetime "period_start"
    t.datetime "period_end"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["stripe_invoice_id"], name: "index_invoices_on_stripe_invoice_id", unique: true
    t.index ["user_id"], name: "index_invoices_on_user_id"
  end

  create_table "sessions", force: :cascade do |t|
    t.integer "user_id", null: false
    t.string "ip_address"
    t.string "user_agent"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["user_id"], name: "index_sessions_on_user_id"
  end

  create_table "subscriptions", force: :cascade do |t|
    t.integer "user_id", null: false
    t.string "stripe_subscription_id", null: false
    t.string "stripe_price_id", default: "", null: false
    t.string "plan_name", default: "", null: false
    t.string "status", default: "active", null: false
    t.datetime "current_period_end"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["stripe_subscription_id"], name: "index_subscriptions_on_stripe_subscription_id", unique: true
    t.index ["user_id"], name: "index_subscriptions_on_user_id"
  end

  create_table "users", force: :cascade do |t|
    t.string "email_address", null: false
    t.string "password_digest", null: false
    t.string "first_name", default: "", null: false
    t.string "last_name", default: "", null: false
    t.string "stripe_customer_id", default: "", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["email_address"], name: "index_users_on_email_address", unique: true
  end

  add_foreign_key "invoices", "users"
  add_foreign_key "sessions", "users"
  add_foreign_key "subscriptions", "users"
end
