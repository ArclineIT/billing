class Session < ApplicationRecord
  # Login sessions expire after this duration (matches the cookie lifetime).
  TTL = 30.days

  belongs_to :user

  scope :expired, -> { where(created_at: ...TTL.ago) }
end
