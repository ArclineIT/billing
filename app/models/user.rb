class User < ApplicationRecord
  MIN_PASSWORD_LENGTH = 8

  has_secure_password
  has_many :sessions, dependent: :destroy
  has_many :subscriptions, dependent: :destroy
  has_many :invoices, dependent: :destroy

  normalizes :email_address, with: ->(e) { e.strip.downcase }

  validates :email_address, presence: true, uniqueness: true
  validates :first_name, :last_name, presence: true
  validates :password, length: { minimum: MIN_PASSWORD_LENGTH }, allow_nil: true

  def full_name
    [ first_name, last_name ].compact_blank.join(" ")
  end
end
