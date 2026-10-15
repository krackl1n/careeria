#!/usr/bin/env ruby
# Local adapter for jimmyjazz/IDEF0-SVG. Keep the upstream layout code intact.
require 'cgi'
require 'json'
require 'optparse'
require_relative 'upstream/lib/idef0/cli'
require_relative 'language'
require_relative 'boundary'

module CareeriaIdef0Style
  COLOR = /\A(?:\#[0-9a-fA-F]{3}(?:[0-9a-fA-F]{3})?|[a-zA-Z]+)\z/
  DEFAULT_KEYS = %w[arrow arrow_text block_fill block_stroke block_text].freeze
  BLOCK_KEYS = %w[fill stroke text].freeze
  ARROW_KEYS = %w[stroke text].freeze

  class << self
    attr_accessor :settings

    def validate_colors(values, keys, location)
      raise "#{location} must be an object" unless values.is_a?(Hash)
      values.each do |key, value|
        raise "unknown #{location} field: #{key}" unless keys.include?(key)
        raise "invalid #{location}.#{key} color: #{value.inspect}" unless value.is_a?(String) && COLOR.match?(value)
      end
    end

    def load(path)
      data = path ? JSON.parse(File.read(path)) : {}
      raise 'style file must be an object' unless data.is_a?(Hash)
      unknown = data.keys - %w[mode process views defaults blocks arrows boundaries order branching]
      raise "unknown style fields: #{unknown.join(', ')}" unless unknown.empty?
      if data.key?('order')
        order = data['order']
        raise "invalid order mode: #{order.inspect}" unless %w[auto source fixed heuristic].include?(order)
      end
      if data.key?('branching')
        branching = data['branching']
        valid = [true, false, 'auto', 'true', 'false', 'none', 'mechanisms', 'all']
        raise "invalid branching mode: #{branching.inspect}" unless valid.include?(branching)
      end
      validate_colors(data.fetch('defaults', {}), DEFAULT_KEYS, 'defaults')
      { 'blocks' => BLOCK_KEYS, 'arrows' => ARROW_KEYS }.each do |group, keys|
        entries = data.fetch(group, {})
        raise "#{group} must be an object" unless entries.is_a?(Hash)
        entries.each { |name, values| validate_colors(values, keys, "#{group}.#{name}") }
      end
      CareeriaIdef0Boundary.load(data)
      self.settings = data
    end

    def order_mode
      settings&.fetch('order', 'auto') || 'auto'
    end

    def branching_mode
      settings&.fetch('branching', 'auto')
    end

    def branching_mechanisms?
      mode = branching_mode
      return false if mode == false || mode == 'false' || mode == 'none'
      true
    end

    def branching_controls?
      mode = branching_mode
      return false if mode == false || mode == 'false' || mode == 'none' || mode == 'mechanisms'
      true
    end

    def box_svg(box)
      defaults = settings.fetch('defaults', {})
      style = settings.fetch('blocks', {}).fetch(box.name, {})
      fill = style.fetch('fill', defaults.fetch('block_fill', 'none'))
      stroke = style.fetch('stroke', defaults.fetch('block_stroke', 'black'))
      text = style.fetch('text', defaults.fetch('block_text', 'black'))
      markup = box.to_svg
        .sub("fill='none' stroke='black'", "fill='#{fill}' stroke='#{stroke}'")
        .sub('<text ', "<text fill='#{text}' ")
      "<g data-idef0-block='#{CGI.escapeHTML(box.name)}'>#{markup}</g>"
    end

    def arrow_svg(line)
      defaults = settings.fetch('defaults', {})
      style = settings.fetch('arrows', {}).fetch(line.name, {})
      stroke = style.fetch('stroke', defaults.fetch('arrow', 'black'))
      text = style.fetch('text', defaults.fetch('arrow_text', 'black'))
      markup = line.to_svg
        .gsub("stroke='black'", "stroke='#{stroke}'")
        .gsub("fill='black'", "fill='#{stroke}'")
        .gsub('<text ', "<text fill='#{text}' ")
      attributes, extras = CareeriaIdef0Boundary.decorate(line, stroke)
      "<g data-idef0-arrow='#{CGI.escapeHTML(line.name)}'#{attributes}>#{markup}#{extras}</g>"
    end

    def branched_mechanism_svg(name, lines, bus_index)
      defaults = settings.fetch('defaults', {})
      style = settings.fetch('arrows', {}).fetch(name, {})
      stroke = style.fetch('stroke', defaults.fetch('arrow', 'black'))
      text_color = style.fetch('text', defaults.fetch('arrow_text', 'black'))

      sorted = lines.sort_by(&:x1)
      primary = sorted.first
      x_min = sorted.first.x1
      x_max = sorted.last.x1
      y_bottom = sorted.map(&:y1).max
      boxes_bottom = sorted.map(&:y2).max

      bus_y = boxes_bottom + 15 + bus_index * 20
      trunk = %(<line x1='#{x_min}' y1='#{y_bottom - 20}' x2='#{x_min}' y2='#{bus_y}' stroke='#{stroke}' />)
      bus = %(<line x1='#{x_min}' y1='#{bus_y}' x2='#{x_max}' y2='#{bus_y}' stroke='#{stroke}' />)
      branches = sorted.map do |line|
        x = line.x1
        target_y = line.y2
        %(<line x1='#{x}' y1='#{bus_y}' x2='#{x}' y2='#{target_y}' stroke='#{stroke}' />\n) \
          + %(<polygon fill='#{stroke}' stroke='#{stroke}' points='#{x},#{target_y} #{x - 3},#{target_y + 6} #{x + 3},#{target_y + 6} #{x},#{target_y}' />)
      end.join("\n")

      label_svg = %(<text fill='#{text_color}' text-anchor='middle' x='#{x_min}' y='#{y_bottom - 5}'>#{CGI.escapeHTML(name)}</text>)
      attributes, extras = CareeriaIdef0Boundary.decorate(primary, stroke)

      sorted[1..-1].each do |other_line|
        _other_attr, other_extra = CareeriaIdef0Boundary.decorate(other_line, stroke)
        if other_extra.include?('data-idef0-tunnel-marker')
          extras << other_extra.scan(/<g data-idef0-tunnel-marker='parent'.*?<\/g>/m).join
        end
      end

      markup = "#{trunk}\n#{bus}\n#{branches}\n#{label_svg}"
      "<g data-idef0-arrow='#{CGI.escapeHTML(name)}'#{attributes}>\n#{markup}\n#{extras}\n</g>"
    end

    def branched_guidance_svg(name, lines, bus_index)
      defaults = settings.fetch('defaults', {})
      style = settings.fetch('arrows', {}).fetch(name, {})
      stroke = style.fetch('stroke', defaults.fetch('arrow', 'black'))
      text_color = style.fetch('text', defaults.fetch('arrow_text', 'black'))

      sorted = lines.sort_by(&:x1)
      primary = sorted.first
      x_min = sorted.first.x1
      x_max = sorted.last.x1
      y_top = sorted.map(&:y1).min
      boxes_top = sorted.map(&:y2).min

      bus_y = boxes_top - 15 - bus_index * 20
      trunk = %(<line x1='#{x_min}' y1='#{y_top + 20}' x2='#{x_min}' y2='#{bus_y}' stroke='#{stroke}' />)
      bus = %(<line x1='#{x_min}' y1='#{bus_y}' x2='#{x_max}' y2='#{bus_y}' stroke='#{stroke}' />)
      branches = sorted.map do |line|
        x = line.x1
        target_y = line.y2
        %(<line x1='#{x}' y1='#{bus_y}' x2='#{x}' y2='#{target_y}' stroke='#{stroke}' />\n) \
          + %(<polygon fill='#{stroke}' stroke='#{stroke}' points='#{x},#{target_y} #{x - 3},#{target_y - 6} #{x + 3},#{target_y - 6} #{x},#{target_y}' />)
      end.join("\n")

      label_svg = %(<text fill='#{text_color}' text-anchor='middle' x='#{x_min}' y='#{y_top + 14}'>#{CGI.escapeHTML(name)}</text>)
      attributes, extras = CareeriaIdef0Boundary.decorate(primary, stroke)

      sorted[1..-1].each do |other_line|
        _other_attr, other_extra = CareeriaIdef0Boundary.decorate(other_line, stroke)
        if other_extra.include?('data-idef0-tunnel-marker')
          extras << other_extra.scan(/<g data-idef0-tunnel-marker='parent'.*?<\/g>/m).join
        end
      end

      markup = "#{trunk}\n#{bus}\n#{branches}\n#{label_svg}"
      "<g data-idef0-arrow='#{CGI.escapeHTML(name)}'#{attributes}>\n#{markup}\n#{extras}\n</g>"
    end
  end
end

module CareeriaIdef0Diagram
  def create_lines
    order_mode = CareeriaIdef0Style.order_mode
    if order_mode == 'heuristic'
      super
      return
    end

    source_boxes = @boxes.dup.sequence!
    source_lines = source_boxes.reduce(IDEF0::ArraySet.new) do |lines, target|
      source_boxes.each do |source|
        IDEF0::INTERNAL_LINE_TYPES.each do |line_type|
          line_type.make_line(source, target) { |line| lines.add(line) }
        end
      end
      lines
    end
    source_backward_count = source_lines.count(&:backward?)

    if order_mode == 'source' || order_mode == 'fixed' || source_backward_count.zero?
      @boxes = source_boxes
      @lines = source_lines
    else
      super
      heuristic_backward_count = @lines.count(&:backward?)
      if source_backward_count <= heuristic_backward_count
        @boxes = source_boxes
        @lines = source_lines
      end
    end

    @lines.each(&:attach)

    (IDEF0::EXTERNAL_LINE_TYPES + IDEF0::UNATTACHED_LINE_TYPES).each do |line_type|
      @boxes.each do |box|
        line_type.make_line(self, box) { |line| @lines.add(line.attach) }
      end
    end
  end

  def layout
    super
    if CareeriaIdef0Style.branching_mechanisms?
      mech_groups = @lines.select { |l| l.is_a?(IDEF0::ExternalMechanismLine) }.group_by(&:name)
      branched_count = mech_groups.count { |_, ls| ls.count > 1 }
      if branched_count > 1
        extra_clearance = (branched_count - 1) * 20
        @lines.select { |l| l.is_a?(IDEF0::ExternalMechanismLine) }.each do |line|
          line.instance_eval { add_clearance_from(@target.bottom_side, extra_clearance) }
        end
        resize(right_edge + 20, bottom_edge + 20 + extra_clearance)
      end
    end
  end

  def generate_boxes
    @boxes.map { |box| CareeriaIdef0Style.box_svg(box) }.join("\n")
  end

  def generate_lines
    result = []
    mech_lines = @lines.select { |l| l.is_a?(IDEF0::ExternalMechanismLine) }
    guidance_lines = @lines.select { |l| l.is_a?(IDEF0::ExternalGuidanceLine) }
    other_lines = @lines.reject { |l| l.is_a?(IDEF0::ExternalMechanismLine) || l.is_a?(IDEF0::ExternalGuidanceLine) }

    other_lines.each { |line| result << CareeriaIdef0Style.arrow_svg(line) }

    if CareeriaIdef0Style.branching_controls?
      guidance_groups = guidance_lines.group_by(&:name)
      bus_idx = 0
      guidance_groups.each do |name, lines|
        if lines.count > 1
          result << CareeriaIdef0Style.branched_guidance_svg(name, lines, bus_idx)
          bus_idx += 1
        else
          result << CareeriaIdef0Style.arrow_svg(lines.first)
        end
      end
    else
      guidance_lines.each { |l| result << CareeriaIdef0Style.arrow_svg(l) }
    end

    if CareeriaIdef0Style.branching_mechanisms?
      mech_groups = mech_lines.group_by(&:name)
      bus_idx = 0
      mech_groups.each do |name, lines|
        if lines.count > 1
          result << CareeriaIdef0Style.branched_mechanism_svg(name, lines, bus_idx)
          bus_idx += 1
        else
          result << CareeriaIdef0Style.arrow_svg(lines.first)
        end
      end
    else
      mech_lines.each { |l| result << CareeriaIdef0Style.arrow_svg(l) }
    end

    result.join("\n")
  end
end

IDEF0::Diagram.prepend(CareeriaIdef0Diagram)

if __FILE__ == $PROGRAM_NAME
  options = { mode: 'decompose' }
  OptionParser.new do |parser|
    parser.on('--input PATH') { |value| options[:input] = value }
    parser.on('--style PATH') { |value| options[:style] = value }
    parser.on('--mode MODE') { |value| options[:mode] = value }
    parser.on('--process NAME') { |value| options[:process] = value }
    parser.on('--order ORDER') { |value| options[:order] = value }
    parser.on('--branching BRANCHING') { |value| options[:branching] = value }
  end.parse!

  abort 'missing --input' unless options[:input]
  abort 'mode must be auto, decompose, schematic or focus' unless %w[auto decompose schematic focus].include?(options[:mode])

  begin
    CareeriaIdef0Style.load(options[:style])
    CareeriaIdef0Style.settings['order'] = options[:order] if options[:order]
    CareeriaIdef0Style.settings['branching'] = options[:branching] if options[:branching]
    statements = File.open(options[:input], 'r:utf-8') { |file| IDEF0::Statement.parse(file) }
    root = IDEF0::Process.parse(statements)
    CareeriaIdef0Boundary.validate(root)
    process = options[:process] ? root.find(options[:process]) : root
    abort "process not found: #{options[:process]}" unless process
    mode = options[:mode] == 'auto' ? (process.decomposable? ? 'decompose' : 'focus') : options[:mode]
    CareeriaIdef0Boundary.select_view(process, mode)
    puts process.public_send(mode).to_svg
  rescue StandardError => error
    warn "IDEF0-SVG: #{error.message}"
    exit 1
  end
end
