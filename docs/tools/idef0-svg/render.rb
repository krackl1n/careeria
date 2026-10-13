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
      unknown = data.keys - %w[mode process views defaults blocks arrows boundaries]
      raise "unknown style fields: #{unknown.join(', ')}" unless unknown.empty?
      validate_colors(data.fetch('defaults', {}), DEFAULT_KEYS, 'defaults')
      { 'blocks' => BLOCK_KEYS, 'arrows' => ARROW_KEYS }.each do |group, keys|
        entries = data.fetch(group, {})
        raise "#{group} must be an object" unless entries.is_a?(Hash)
        entries.each { |name, values| validate_colors(values, keys, "#{group}.#{name}") }
      end
      CareeriaIdef0Boundary.load(data)
      self.settings = data
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
  end
end

module CareeriaIdef0Diagram
  def generate_boxes
    @boxes.map { |box| CareeriaIdef0Style.box_svg(box) }.join("\n")
  end

  def generate_lines
    @lines.map { |line| CareeriaIdef0Style.arrow_svg(line) }.join("\n")
  end
end

IDEF0::Diagram.prepend(CareeriaIdef0Diagram)

options = { mode: 'decompose' }
OptionParser.new do |parser|
  parser.on('--input PATH') { |value| options[:input] = value }
  parser.on('--style PATH') { |value| options[:style] = value }
  parser.on('--mode MODE') { |value| options[:mode] = value }
  parser.on('--process NAME') { |value| options[:process] = value }
end.parse!

abort 'missing --input' unless options[:input]
abort 'mode must be auto, decompose, schematic or focus' unless %w[auto decompose schematic focus].include?(options[:mode])

begin
  CareeriaIdef0Style.load(options[:style])
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
