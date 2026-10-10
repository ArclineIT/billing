class Subscription < ApplicationRecord
  belongs_to :user

  enum :status, {
    active: "active",
    past_due: "past_due",
    cancelling: "cancelling",
    cancelled: "cancelled"
  }, default: "active"

  validates :stripe_subscription_id, presence: true, uniqueness: true

  scope :current, -> { order(created_at: :desc) }
  scope :billable, -> { where(status: %w[active cancelling]) }
end
