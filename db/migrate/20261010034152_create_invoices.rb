class CreateInvoices < ActiveRecord::Migration[8.1]
  def change
    create_table :invoices do |t|
      t.references :user, null: false, foreign_key: true
      t.string :stripe_invoice_id, null: false
      t.integer :amount_cents, null: false, default: 0
      t.string :currency, null: false, default: "usd"
      t.string :status, null: false, default: "open"
      t.string :invoice_pdf_url, null: false, default: ""
      t.datetime :period_start
      t.datetime :period_end

      t.timestamps
    end
    add_index :invoices, :stripe_invoice_id, unique: true
  end
end
