class User < ApplicationRecord
  has_secure_password

  validates :username, :email, presence: true, uniqueness: true

  before_validation do
    if self.email.present?
      self.email = self.email.downcase
    end
  end
end
