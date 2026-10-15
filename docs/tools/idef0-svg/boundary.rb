require 'cgi'
require 'set'

# IDEF0 boundary correspondence around the vendored layout engine. The engine
# creates external lines, but does not validate parent/child interfaces or mark
# ICOM codes and tunnels. Keep those rules in this adapter.
module CareeriaIdef0Boundary
  SIDE_KIND = {
    left_side: 'input', top_side: 'control',
    right_side: 'output', bottom_side: 'mechanism'
  }.freeze
  KIND_SIDE = SIDE_KIND.invert.freeze
  CODE = { 'input' => 'I', 'control' => 'C', 'output' => 'O', 'mechanism' => 'M' }.freeze

  class << self
    def load(settings)
      raw = settings.fetch('boundaries', {})
      raise 'boundaries must be an object' unless raw.is_a?(Hash)
      @tunnels = {}
      raw.each do |process_name, entry|
        raise 'boundary process names must be non-empty text' unless process_name.is_a?(String) && !process_name.strip.empty?
        raise "boundaries.#{process_name} must be an object" unless entry.is_a?(Hash)
        unknown = entry.keys - %w[parent_tunnels child_tunnels]
        raise "unknown boundaries.#{process_name} fields: #{unknown.join(', ')}" unless unknown.empty?
        @tunnels[process_name] = {
          parent: parse_pairs(entry.fetch('parent_tunnels', {}), "#{process_name}.parent_tunnels"),
          child: parse_pairs(entry.fetch('child_tunnels', {}), "#{process_name}.child_tunnels")
        }
      end
    end

    def parse_pairs(values, location)
      raise "#{location} must be an object" unless values.is_a?(Hash)
      pairs = Set.new
      values.each do |kind, names|
        raise "#{location}: unknown ICOM kind #{kind.inspect}" unless KIND_SIDE.key?(kind)
        raise "#{location}.#{kind} must be a list" unless names.is_a?(Array)
        names.each do |name|
          raise "#{location}.#{kind}: names must be non-empty text" unless name.is_a?(String) && !name.strip.empty?
          pair = [kind, name]
          raise "#{location}: duplicate #{kind}:#{name}" if pairs.include?(pair)
          pairs.add(pair)
        end
      end
      pairs
    end

    def children(process)
      process.instance_variable_get(:@children)
    end

    def external?(line)
      line.source.is_a?(IDEF0::Diagram) || line.target.is_a?(IDEF0::Diagram)
    end

    def incoming_kind(line)
      name = line.class.name
      return 'control' if name.include?('Guidance')
      return 'mechanism' if name.include?('Mechanism')
      return 'input' if name.include?('Input')
      raise "unknown incoming line type: #{name}"
    end

    def boundary_pair(line)
      [line.target.is_a?(IDEF0::Diagram) ? 'output' : incoming_kind(line), line.name]
    end

    def interface(process)
      pairs = Set.new
      process.each_dependency { |side, name| pairs.add([SIDE_KIND.fetch(side), name]) }
      pairs
    end

    def format_pairs(pairs)
      pairs.map { |kind, name| "#{kind}:#{name}" }.sort.join(', ')
    end

    def validate(root)
      @processes = {}
      @diagrams = {}
      @parents = {}
      queue = [[root, nil]]
      until queue.empty?
        process, parent = queue.shift
        raise "duplicate process: #{process.name}" if @processes.key?(process.name)
        @processes[process.name] = process
        @parents[process.name] = parent
        @diagrams[process.name] = process.decompose unless children(process).empty?
        children(process).each { |child| queue << [child, process] }
      end
      unknown = @tunnels.keys - @processes.keys
      raise "boundaries refer to unknown processes: #{unknown.join(', ')}" unless unknown.empty?
      without_children = @tunnels.keys - @diagrams.keys
      raise "boundaries require a decomposed process: #{without_children.join(', ')}" unless without_children.empty?
      if @tunnels.fetch(root.name, {})[:parent]&.any?
        raise "#{root.name}: parent_tunnels on the root process cannot be marked on a parent box"
      end

      @diagrams.each do |name, diagram|
        expected = interface(@processes.fetch(name))
        actual = Set.new
        diagram.instance_variable_get(:@lines).each do |line|
          actual.add(boundary_pair(line)) if external?(line)
        end
        tunnel = @tunnels.fetch(name, { parent: Set.new, child: Set.new })
        missing = expected - actual
        extra = actual - expected
        unless missing == tunnel[:parent] && extra == tunnel[:child]
          raise "#{name}: boundary mismatch; missing on child=[#{format_pairs(missing)}], " \
                "unmatched child boundary=[#{format_pairs(extra)}]; " \
                "declare exactly these in parent_tunnels and child_tunnels"
        end
      end
      @codes = {}
      @processes.each_value { |process| @codes[process.name] = codes_for(process, root) }
    end

    def codes_for(process, root)
      parent = @parents.fetch(process.name)
      diagram = parent ? @diagrams.fetch(parent.name) : root.focus
      box = diagram.instance_variable_get(:@boxes).find { |item| item.name == process.name }
      raise "box missing for #{process.name}" unless box
      result = {}
      CODE.each do |kind, letter|
        anchors = []
        box.public_send(KIND_SIDE.fetch(kind)).each_anchor { |anchor| anchors << anchor }
        anchors.sort_by(&:sequence).each_with_index do |anchor, index|
          result[[kind, anchor.name]] = "#{letter}#{index + 1}"
        end
      end
      result
    end

    def select_view(process, mode)
      @current_process = process.name
      @current_mode = mode
    end

    def tunnel_marker(x, y, vertical, color, location)
      rotation = vertical ? ' rotate(90)' : ''
      %(<g data-idef0-tunnel-marker='#{location}' transform='translate(#{x} #{y})#{rotation}'>) \
        + %(<path d='M -5 -8 Q -11 0 -5 8 M 5 -8 Q 11 0 5 8' fill='none' stroke='#{color}' stroke-width='1.8' />) \
        + '</g>'
    end

    def icom_label(line, code, color)
      kind = boundary_pair(line).first
      case kind
      when 'input' then x, y, anchor = line.x1 - 4, line.y1 + 4, 'end'
      when 'output' then x, y, anchor = line.x2 + 4, line.y2 + 4, 'start'
      when 'control' then x, y, anchor = line.x1 + 13, line.y1 - 4, 'start'
      else x, y, anchor = line.x1 + 13, line.y1 - 24, 'start'
      end
      %(<text class='idef0-icom' x='#{x}' y='#{y}' text-anchor='#{anchor}' fill='#{color}' font-size='11' font-weight='700'>#{code}</text>)
    end

    def child_marker(line, color)
      kind = boundary_pair(line).first
      case kind
      when 'input' then tunnel_marker(line.x1 + 14, line.y1, false, color, 'child')
      when 'output' then tunnel_marker(line.x2 - 14, line.y2, false, color, 'child')
      when 'control' then tunnel_marker(line.x1, line.y1 + 34, true, color, 'child')
      else tunnel_marker(line.x1, line.y1 - 34, true, color, 'child')
      end
    end

    def parent_markers(line, color)
      markers = []
      names = []
      if line.source.is_a?(IDEF0::ProcessBox)
        name = line.source.name
        if @tunnels.fetch(name, {})[:parent]&.include?(['output', line.name])
          names << name
          markers << tunnel_marker(line.x1 + 14, line.y1, false, color, 'parent')
        end
      end
      if line.target.is_a?(IDEF0::ProcessBox)
        name = line.target.name
        kind = incoming_kind(line)
        if @tunnels.fetch(name, {})[:parent]&.include?([kind, line.name])
          names << name
          markers << case kind
                     when 'input' then tunnel_marker(line.x2 - 14, line.y2, false, color, 'parent')
                     when 'control' then tunnel_marker(line.x2, line.y2 - 14, true, color, 'parent')
                     else tunnel_marker(line.x2, line.y2 + 14, true, color, 'parent')
                     end
        end
      end
      [names, markers.join]
    end

    def decorate(line, color)
      return ['', ''] unless @current_mode == 'decompose'
      attributes = []
      extra = ''
      if external?(line)
        pair = boundary_pair(line)
        tunnel = @tunnels.fetch(@current_process, { child: Set.new })
        if tunnel[:child].include?(pair)
          attributes << "data-idef0-tunnel='child'"
          extra << child_marker(line, color)
        else
          code = @codes.fetch(@current_process).fetch(pair)
          attributes << "data-idef0-icom='#{code}'"
          extra << icom_label(line, code, color)
        end
      end
      names, markers = parent_markers(line, color)
      unless names.empty?
        attributes << "data-idef0-tunnel-parent='#{CGI.escapeHTML(names.uniq.join(', '))}'"
        extra << markers
      end
      [attributes.empty? ? '' : ' ' + attributes.join(' '), extra]
    end
  end
end
