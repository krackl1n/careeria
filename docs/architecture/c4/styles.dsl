styles {
    element "Element" {
        shape RoundedBox
        background #64748b
        color #ffffff
        stroke #475569
    }

    element "Person" {
        shape Person
        background #334155
        color #ffffff
    }

    element "Software System" {
        background #2563eb
        color #ffffff
    }

    element "ExternalSystem" {
        background #64748b
        color #ffffff
        border dashed
    }

    element "Frontend" {
        shape WebBrowser
        background #2563eb
        color #ffffff
    }

    element "Edge" {
        background #0ea5e9
        color #ffffff
    }

    element "Gateway" {
        background #0284c7
        color #ffffff
        strokeWidth 4
    }

    element "BusinessService" {
        background #16a34a
        color #ffffff
    }

    element "PlatformService" {
        background #0891b2
        color #ffffff
    }

    element "Authorization" {
        background #7c3aed
        color #ffffff
    }

    element "EventBus" {
        shape Pipe
        background #f59e0b
        color #111827
        stroke #b45309
        strokeWidth 4
    }

    element "EventInfrastructure" {
        background #d97706
        color #ffffff
    }

    element "Realtime" {
        background #db2777
        color #ffffff
        stroke #9d174d
        strokeWidth 4
    }

    element "Media" {
        background #9333ea
        color #ffffff
        stroke #6b21a8
        strokeWidth 4
    }

    element "Database" {
        shape Cylinder
        background #475569
        color #ffffff
    }

    element "ObjectStorage" {
        shape Bucket
        background #475569
        color #ffffff
    }

    relationship "Relationship" {
        color #64748b
        style solid
        thickness 2
        routing Direct
        fontSize 18
        width 180
    }

    relationship "Async" {
        color #d97706
        style dashed
        thickness 2
    }

    relationship "RealtimeFlow" {
        color #db2777
        thickness 3
    }

    relationship "MediaFlow" {
        color #9333ea
        thickness 3
    }

    relationship "CDC" {
        color #a16207
        style dotted
        thickness 2
    }
}
