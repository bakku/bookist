# frozen_string_literal: true

# PasswordResetAuthority defines the logic for resetting passwords of resettable records, e.g. users.
class PasswordResetAuthority
  DEFAULT_TOKEN_VALIDITY_DURATION = 1.hour

  # Configure the authority you want to use.
  #
  # @param token_validity_duration [ActiveSupport::Duration] the validity duration of the password_reset_token,
  #   by default 1 hour
  def initialize(token_validity_duration = DEFAULT_TOKEN_VALIDITY_DURATION)
    @token_validity_duration = token_validity_duration
  end

  # Returns true in case the password_reset_token is not older than the validity duration, otherwise false.
  #
  # @param resettable_record [ApplicationRecord] the record which should be checked regarding the possibility of
  #   resetting its password
  def resettable?(resettable_record)
    Time.current - @token_validity_duration < resettable_record.password_reset_token_created_at
  end

  # Assigns a new password_reset_token and sets the password_reset_token_created_at timestamp.
  #
  # @param resettable_record [ApplicationRecord] the record for which the password reset should be prepared
  def prepare_reset!(resettable_record)
    resettable_record.update(password_reset_token: SecureRandom.uuid, password_reset_token_created_at: Time.current)
  end

  # Performs the password reset.
  #
  # @param resettable_record [ApplicationRecord] the record for which the password reset should be performed
  # @param new_password [String] the new password for the record
  # @param new_password_confirmation [String] the password confirmation of the new password for the record
  def complete_reset!(resettable_record, new_password, new_password_confirmation)
    return false unless resettable?(resettable_record)

    resettable_record.update(
      password: new_password,
      password_confirmation: new_password_confirmation,
      password_reset_token: nil,
      password_reset_token_created_at: nil
    )
  end
end
