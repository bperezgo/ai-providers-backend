-- 001_seed_businesses.sql
-- Seeds the businesses table with laguna-escondida and vientre-vivo.
-- Uses ON CONFLICT to upsert (update if slug already exists).

INSERT INTO businesses (id, name, slug, type, location, description, discovery, created_at, updated_at)
VALUES (
    '019560a0-0001-7000-8000-000000000001',
    'Laguna Escondida',
    'laguna-escondida',
    'restaurant',
    'Vereda La Primavera, Marinilla, Antioquia, Colombia',
    'Restaurante campestre con pesca deportiva. Experiencia campestre donde las personas pueden pescar, comer, o ambas. Infraestructura en guadua con aspecto rústico, zona verde, lago de ~50m x 15m.',
    '{
        "date": "2026-02-02",
        "status": "active",
        "summary": "Restaurante campestre con pesca deportiva ubicado en zona rural de Marinilla. Opera hace 5 meses como marca renovada (anteriormente D''ely pez). Rating 4.6 en Google Maps. Diferenciador: variedad de 5-6 especies de peces cuando la competencia típicamente solo ofrece trucha.",
        "offer": {
            "restaurant": {
                "categories": ["Pescados (Yamú frito, Tilapia, Trucha)", "Otras carnes", "Comida rápida (Hamburguesas)", "Menú infantil", "Bebidas"]
            },
            "sport_fishing": {
                "species": [
                    {"name": "Tilapia Roja", "availability": "alta"},
                    {"name": "Tilapia Nilótica", "availability": "alta"},
                    {"name": "Yamú", "availability": "alta"},
                    {"name": "Doradas", "availability": "alta"},
                    {"name": "Cachamas", "availability": "alta"},
                    {"name": "Kodi", "availability": "baja"}
                ]
            },
            "facilities": ["Zona verde", "Espacio para niños", "Parqueadero", "Espacios semi-privados", "1 kiosco privado", "~30 configuraciones de mesas (60 individuales)"]
        },
        "metrics": {
            "months_operating": 5,
            "clients_per_week": 30,
            "best_days": ["Domingos", "Festivos"],
            "avg_ticket_cop": {"min": 50000, "max": 70000},
            "google_rating": 4.6,
            "google_reviews": 12,
            "instagram_followers": 300,
            "facebook_followers": 100
        },
        "customers": {
            "profile": "Familias con niños, parejas, grupos de amigos. Edades: niños de todas las edades + adultos hasta 40 años. Procedencia: Marinilla, Rionegro, Medellín.",
            "segments": [
                {"name": "Familias con niños", "motivation": "Lugar seguro, actividad para niños, almuerzo"},
                {"name": "Parejas", "motivation": "Ambiente campestre, tranquilidad"},
                {"name": "Pescadores casuales", "motivation": "Pescar como actividad complementaria"},
                {"name": "Pescadores intermedios", "motivation": "Variedad de especies, experiencia de pesca"},
                {"name": "Celebraciones", "motivation": "Espacio privado, ambiente, servicio"}
            ],
            "drivers": ["Ambiente campestre", "Opción de pesca", "Comida"],
            "acquisition": ["Redes sociales (Instagram, Facebook)", "Pasan por el lugar", "Boca a boca"]
        },
        "differentiators": {
            "primary": "Variedad de 5-6 especies de peces (vs competencia que solo ofrece trucha)",
            "secondary": ["Punto medio de pesca (no extrema, no solo restaurante)", "Infraestructura en guadua (aspecto rústico auténtico)", "Ubicación rural (experiencia campestre genuina)"]
        },
        "competitors": [
            {"name": "Alcaravanes", "location": "Marinilla", "strength": "Grande, múltiples atracciones"},
            {"name": "PsiciLagos", "location": "Zona", "strength": "Tiene piscina"},
            {"name": "Truchera La Honda", "location": "Guarne", "strength": "Ambiente campestre"}
        ],
        "goals": {
            "timeframe": "6 meses",
            "families_per_week": {"current": 30, "target": 100},
            "avg_ticket_cop": {"current": 60000, "target": 75000},
            "weekly_revenue_cop": {"current": 1800000, "target": 7500000},
            "google_reviews": {"current": 12, "target": 50}
        },
        "budget": {
            "marketing_monthly_cop": {"min": 500000, "max": 1000000},
            "marketing_hours_per_week": 20,
            "responsible": "Propietario"
        },
        "positioning": "El único lugar de pesca con 5 especies de peces en el Oriente Antioqueño",
        "vision": "Laguna Escondida será reconocida como un destino de pesca variada donde los pescadores vienen a experimentar diferentes tipos de peces, con la tilapia como plato principal. Será un lugar campestre, agradable, familiar y pet-friendly, ideal para celebrar cumpleaños y eventos especiales."
    }'::jsonb,
    NOW(),
    NOW()
)
ON CONFLICT (slug) DO UPDATE SET
    name        = EXCLUDED.name,
    type        = EXCLUDED.type,
    location    = EXCLUDED.location,
    description = EXCLUDED.description,
    discovery   = EXCLUDED.discovery,
    updated_at  = NOW();

INSERT INTO businesses (id, name, slug, type, location, description, discovery, created_at, updated_at)
VALUES (
    '019560a0-0002-7000-8000-000000000002',
    'Vientre Vivo',
    'vientre-vivo',
    'course',
    'Global — Hispanohablantes',
    'Proyecto educativo y transformador para madres gestantes hispanohablantes. Acompañamiento del embarazo desde cuatro dimensiones: neuroemocional, corporal-biológica, nutricional y transpersonal.',
    '{
        "date": "2026-03-08",
        "status": "pre-launch",
        "summary": "Proyecto educativo digital para madres gestantes hispanohablantes. Propuesta central: acompañar el embarazo desde 4 dimensiones integrales. Producto inicial: Psicogestar — La Conciencia del Amor (11 módulos, ~5h, $20 USD). Oportunidad clave: enfoque neuroemocional con respaldo científico casi inexplorado en español.",
        "structure": {
            "axes": [
                {"name": "Psicogestar (Neuroemocional)", "focus": "Desarrollo neuroemocional de la madre — conciencia, emociones, vínculo bebé", "status": "active"},
                {"name": "Corporal-Biológico", "focus": "Fisiología del embarazo, movimiento, cuerpo", "status": "future"},
                {"name": "Nutricional", "focus": "Nutrición durante la gestación", "status": "future"},
                {"name": "Transpersonal", "focus": "Espiritualidad, identidad, rito de paso", "status": "future"}
            ]
        },
        "product": {
            "name": "Psicogestar: La Conciencia del Amor",
            "modules": 11,
            "duration_hours": 5,
            "format": ["Audiovisual (video)", "Ebook interactivo"],
            "platform": "Hotmart",
            "price_usd": 20,
            "status": "in_production",
            "completion_pct": 50,
            "planned_launch": "mid-2026"
        },
        "team": [
            {"role": "Creador de contenido / autoridad", "title": "Neuropsicólogo", "responsibility": "Diseño curricular, credibilidad científica"},
            {"role": "Marketing", "title": "Bryan", "responsibility": "Estrategia, contenido en redes, crecimiento"},
            {"role": "Recurso técnico", "title": "Desarrollador de software", "responsibility": "Plataforma, integraciones técnicas"}
        ],
        "target_customer": {
            "who": "Madres gestantes hispanohablantes",
            "age": {"min": 16, "max": 40},
            "geography": "Global — habla hispana (Latinoamérica + España)",
            "pregnancy_stage": "Cualquier etapa",
            "parity": "Primíparas y madres con embarazos previos",
            "psychographics": [
                "Siente que el sistema médico ignora su dimensión emocional",
                "Busca algo más allá del preparto convencional",
                "Quiere entender qué siente su bebé y cómo sus emociones lo afectan",
                "Vive el embarazo con ansiedad o sensación de soledad",
                "Valora el respaldo científico"
            ],
            "pains": [
                "Ansiedad y confusión emocional durante el embarazo",
                "Falta de atención emocional en el sistema de salud",
                "Desconexión con el bebé",
                "Información superficial o contradictoria",
                "Ausencia de acompañamiento integral"
            ]
        },
        "differentiators": {
            "primary": "Enfoque neuroemocional con base científica — casi inexistente en el mercado hispanohablante",
            "secondary": [
                "Autoridad de neuropsicólogo certificado",
                "Formato dual: audiovisual + ebook interactivo",
                "Visión integral de 4 ejes",
                "Precio accesible de entrada ($20)"
            ]
        },
        "positioning": "El primer programa neuroemocional para madres gestantes hispanohablantes — con respaldo científico, diseñado por un neuropsicólogo — para que vivas tu embarazo con conciencia, conexión y herramientas reales.",
        "competitors": {
            "status": "not_investigated",
            "indirect": [
                "Libros de embarazo",
                "Cursos de preparto de hospitales/clínicas",
                "Doulas y parteras con contenido online",
                "Cuentas de Instagram/TikTok de maternidad",
                "Aplicaciones de embarazo (Ovia, The Bump)"
            ]
        },
        "digital_presence": {
            "instagram": "not_created",
            "tiktok": "not_created",
            "facebook": "not_created",
            "hotmart": "50%",
            "website": "none",
            "email_list": "none"
        },
        "brand": {
            "logo": "not_created",
            "palette": "not_defined",
            "typography": "not_defined",
            "tone": "Cálido, íntimo, científico, esperanzador",
            "visual_refs": "Blancos suaves, verde salvia, beige dorado, luz ámbar"
        },
        "goals": {
            "launch_sales": {"min": 100, "max": 200},
            "launch_revenue_usd": {"min": 2000, "max": 4000},
            "annual_target_usd": 20000,
            "pricing_strategy": "$20 lead magnet → escalar precio según respuesta → más cursos / membresía"
        },
        "budget": {
            "monthly_usd": {"min": 75, "max": 150},
            "condition": "Se activa cuando curso esté listo y redes con tracción",
            "responsible": "Bryan"
        },
        "markets": [
            {"name": "Hispanohablante global", "priority": "alta", "reason": "Idioma nativo, mercado natural"},
            {"name": "Anglófono", "priority": "media", "reason": "Mayor capacidad de pago, más competencia"}
        ],
        "opportunities": [
            "Pioneros en nicho neuroemocional en español",
            "TikTok como canal de adquisición orgánica",
            "Hotmart como ecosistema (afiliados, comunidad)",
            "Escalabilidad: de 1 eje a 4",
            "Mercado anglófono como upside"
        ]
    }'::jsonb,
    NOW(),
    NOW()
)
ON CONFLICT (slug) DO UPDATE SET
    name        = EXCLUDED.name,
    type        = EXCLUDED.type,
    location    = EXCLUDED.location,
    description = EXCLUDED.description,
    discovery   = EXCLUDED.discovery,
    updated_at  = NOW();
