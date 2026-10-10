class Invoice < ApplicationRecord
  belongs_to :user

  enum :status, {
    open: "open",
    paid: "paid",
    void: "void",
    uncollectible: "uncollectible"
  }, default: "open"

  validates :stripe_invoice_id, presence: true, uniqueness: true

  scope :recent, -> { order(created_at: :desc) }

  def amount_display
    dollars = amount_cents / 100
    cents = amount_cents % 100
    code = currency.to_s.upcase

    if code.empty? || code == "USD"
      format("$%d.%02d", dollars, cents)
    else
      format("%d.%02d %s", dollars, cents, code)
    end
  end
end
