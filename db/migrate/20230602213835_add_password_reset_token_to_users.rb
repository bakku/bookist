# frozen_string_literal: true

class AddPasswordResetTokenToUsers < ActiveRecord::Migration[7.0]
  def change
    change_table :users, bulk: true do |t|
      t.text :password_reset_token
      t.column :password_reset_token_created_at, :datetime
    end

    add_index :users, :password_reset_token
  end
end
