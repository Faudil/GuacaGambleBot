HOUSES = {
    "cardboard_box": {
        "id": "cardboard_box",
        "price": 500,
        "max_level": 1,
        "income_per_hour": 5,
        "inventory_bonus": 5,
        "pet_slots_bonus": 1,
        "bank_capacity": 500,
        "crafting_discount": 0,
        "color": 0xB9936C,
        "buffs": ["+5 Inventory Slots", "+1 Pet Slot", "$500 Bank Cap"]
    },
    "wooden_shack": {
        "id": "wooden_shack",
        "price": 5000,
        "max_level": 3,
        "income_per_hour": 50,
        "inventory_bonus": 20,
        "pet_slots_bonus": 2,
        "bank_capacity": 1000,
        "crafting_discount": 0.05,
        "color": 0xA1887F,
        "buffs": ["+20 Inventory Slots", "+2 Pet Slots", "$10,000 Bank Cap", "5% Crafting Discount"]
    },
    "brick_house": {
        "id": "brick_house",
        "price": 25000,
        "max_level": 5,
        "income_per_hour": 250,
        "inventory_bonus": 50,
        "pet_slots_bonus": 5,
        "bank_capacity": 2000,
        "crafting_discount": 0.10,
        "color": 0xD32F2F,
        "buffs": ["+50 Inventory Slots", "+5 Pet Slots", "$2000 Bank Cap", "10% Crafting Discount"]
    },
    "mansion": {
        "id": "mansion",
        "price": 100000,
        "max_level": 10,
        "income_per_hour": 1000,
        "inventory_bonus": 100,
        "pet_slots_bonus": 10,
        "bank_capacity": 5000,
        "crafting_discount": 0.20,
        "color": 0x1E88E5,
        "buffs": ["+100 Inventory Slots", "+10 Pet Slots", "$250,000 Bank Cap", "20% Crafting Discount"]
    },
    "gilded_palace": {
        "id": "gilded_palace",
        "price": 500000,
        "max_level": 20,
        "income_per_hour": 5000,
        "inventory_bonus": 250,
        "pet_slots_bonus": 25,
        "bank_capacity": 1000000,
        "crafting_discount": 0.30,
        "color": 0xFFB300,
        "buffs": ["+250 Inventory Slots", "+25 Pet Slots", "$1,000,000 Bank Cap", "30% Crafting Discount"]
    }
}

# Production de base par type de maison (par heure)
BASE_PRODUCTION = {
    "cardboard_box": {"blé": 0.1},
    "wooden_shack": {"blé": 0.5, "avoine": 0.2},
    "brick_house": {"minerai de fer": 0.5, "charbon": 1.0},
    "mansion": {"minerai d'argent": 0.5, "pépite d'or": 0.2},
    "gilded_palace": {"platine": 0.2, "émeraude": 0.1}
}

# Arbre d'améliorations
UPGRADES_TREE = {
    # BRANCHE MARCHANDE (Argent & Banque)
    "merchant_office": {
        "id": "merchant_office",
        "name": "Bureau de Négociant",
        "branch": "merchant",
        "cost_money": 5000,
        "cost_items": {"charbon": 20, "minerai de cuivre": 10},
        "time_hours": 4,
        "requires": None,
        "bonus_desc": "+20% Capacité Banque, +15% Revenus"
    },
    "merchant_vault": {
        "id": "merchant_vault",
        "name": "Chambre Forte",
        "branch": "merchant",
        "cost_money": 25000,
        "cost_items": {"pépite d'or": 5, "minerai d'argent": 20},
        "time_hours": 24,
        "requires": "merchant_office",
        "bonus_desc": "Capacité Banque doublée"
    },

    # BRANCHE INDUSTRIELLE (Production de ressources)
    "industrial_workshop": {
        "id": "industrial_workshop",
        "name": "Atelier Industriel",
        "branch": "industrial",
        "cost_money": 4000,
        "cost_items": {"caillou": 100, "minerai de fer": 20},
        "time_hours": 6,
        "requires": None,
        "bonus_desc": "Production de ressources x2"
    },
    "industrial_drill": {
        "id": "industrial_drill",
        "name": "Foreuse Automatique",
        "branch": "industrial",
        "cost_money": 30000,
        "cost_items": {"platine": 2, "minerai de fer": 100},
        "time_hours": 48,
        "requires": "industrial_workshop",
        "bonus_desc": "Donne parfois des minerais rares (Diamant, Émeraude)"
    },

    # BRANCHE MYSTIQUE (Bonus & Pets)
    "mystic_altar": {
        "id": "mystic_altar",
        "name": "Autel Mystique",
        "branch": "mystic",
        "cost_money": 7500,
        "cost_items": {"poisson-globe": 5, "plante pourrie": 20},
        "time_hours": 12,
        "requires": None,
        "bonus_desc": "-5% Coûts de Craft, Régénération Pet"
    },
    "mystic_laboratory": {
        "id": "mystic_laboratory",
        "name": "Laboratoire d'Alchimie",
        "branch": "mystic",
        "cost_money": 50000,
        "cost_items": {"émeraude": 5, "potion d'oubli": 1},
        "time_hours": 72,
        "requires": "mystic_altar",
        "bonus_desc": "-15% Coûts de Craft, Chance XP Pet augmentée"
    }
}
