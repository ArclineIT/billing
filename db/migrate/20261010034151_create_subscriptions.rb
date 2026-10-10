class CreateSubscriptions < ActiveRecord::Migration[8.1]
  def change
    create_table :subscriptions do |t|
      t.references :user, null: false, foreign_key: true
      t.string :stripe_subscription_id, null: false
      t.string :stripe_price_id, null: false, default: ""
      t.string :plan_name, null: false, default: ""
      t.string :status, null: false, default: "active"
      t.datetime :current_period_end

      t.timestamps
    end
    add_index :subscriptions, :stripe_subscription_id, unique: true
  end
end
