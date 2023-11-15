# frozen_string_literal: true

class PreparePasswordResetJob
  include Sidekiq::Job

  def perform(email, locale)
    user = User.find_by(email:)

    return if user.nil?

    authority = PasswordResetAuthority.new
    authority.prepare_reset!(user)

    UserMailer.reset_password_instructions(user, locale).deliver_now
  end
end
