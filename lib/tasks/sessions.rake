namespace :sessions do
  desc "Delete login sessions older than Session::TTL"
  task purge: :environment do
    deleted = Session.expired.delete_all
    puts "Purged #{deleted} expired sessions."
  end
end
