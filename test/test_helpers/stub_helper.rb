module StubHelper
  # Temporarily replaces +object.method_name+ with +replacement+ (a value, or a
  # callable invoked with the original arguments) for the duration of the block,
  # restoring the original method afterwards.
  #
  # Minitest 6 no longer ships minitest/mock, so we provide the small amount of
  # stubbing the billing tests need.
  def stub_method(object, method_name, replacement = nil, &block)
    replacement = block if replacement.nil? && block_given?
    raise ArgumentError, "provide a replacement value or block" if replacement.nil?

    singleton = object.singleton_class
    backup = :"__stub_backup_#{method_name}"
    singleton.send(:alias_method, backup, method_name)

    singleton.send(:define_method, method_name) do |*args, **kwargs, &blk|
      if replacement.respond_to?(:call)
        replacement.call(*args, **kwargs, &blk)
      else
        replacement
      end
    end

    yield
  ensure
    singleton.send(:remove_method, method_name)
    singleton.send(:alias_method, method_name, backup)
    singleton.send(:remove_method, backup)
  end
end
