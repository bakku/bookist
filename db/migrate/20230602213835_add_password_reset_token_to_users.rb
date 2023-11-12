class AddPasswordResetTokenToUsers < ActiveRecord::Migration[7.0]
  def change
    add_column :users, :password_reset_token, :text
    add_column :users, :password_reset_token_created_at, :datetime

    add_index :users, :password_reset_token
  end
end
