# Represents a request for a password reset which is used
# in the PasswordResetRequestController.
class PasswordResetRequest
  include ActiveModel::Model

  attr_accessor :email

  validates :email, presence: true
end
