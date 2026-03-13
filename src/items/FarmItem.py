from src.items.Item import ItemRarity
from src.items.ResourceItem import ResourceItem


class RottenPlant(ResourceItem):
    def __init__(self):
        super().__init__(
            "Plante Pourrie", 0, "Tu as mal géré ta ferme...", ItemRarity.common,
            {"stat": "max_hp", "amount": 0}
        )

class Wheat(ResourceItem):
    def __init__(self):
        super().__init__(
            "Blé", 5, "Indispensable pour faire du pain.", ItemRarity.common,
            {"stat": "max_hp", "amount": 2}
        )

class Oat(ResourceItem):
    def __init__(self):
        super().__init__(
            "Avoine", 8, "Parfait pour le petit déjeuner.", ItemRarity.common,
            {"stat": "speed", "amount": 2}
        )

class Corn(ResourceItem):
    def __init__(self):
        super().__init__(
            "Maïs", 12, "Fait aussi du pop-corn !", ItemRarity.common,
            {"stat": "max_hp", "amount": 2}
        )

class Potato(ResourceItem):
    def __init__(self):
        super().__init__(
            "Patate", 20, "On peut en faire de la vodka...", ItemRarity.rare,
            {"stat": "defense", "amount": 2}
        )

class Tomato(ResourceItem):
    def __init__(self):
        super().__init__(
            "Tomate", 25, "Un fruit ou un légume ? Le débat continue.", ItemRarity.rare,
            {"stat": "max_hp", "amount": 3}
        )

class Pumpkin(ResourceItem):
    def __init__(self):
        super().__init__(
            "Citrouille", 40, "Parfait pour Halloween.", ItemRarity.rare,
            {"stat": "defense", "amount": 2}
        )

class CoffeeBean(ResourceItem):
    def __init__(self):
        super().__init__(
            "Grain de Café", 60, "L'or noir du matin.", ItemRarity.epic,
            {"stat": "speed", "amount": 2}
        )

class CocoaBean(ResourceItem):
    def __init__(self):
        super().__init__(
            "Fève de Cacao", 75, "L'ingrédient principal du bonheur (chocolat).", ItemRarity.epic,
            {"stat": "acc", "amount": 2}
        )

class Strawberry(ResourceItem):
    def __init__(self):
        super().__init__(
            "Fraise", 90, "Rouge, sucrée et juteuse.", ItemRarity.epic,
            {"stat": "crit_c", "amount": 2}
        )

class GoldenApple(ResourceItem):
    def __init__(self):
        super().__init__(
            "Pomme Dorée", 150, "Elle brille d'une lueur magique.", ItemRarity.epic,
            {"stat": "max_hp", "amount": 5}
        )

class StarFruit(ResourceItem):
    def __init__(self):
        super().__init__(
            "Fruit Étoile", 250, "Un fruit cosmique d'une autre dimension.", ItemRarity.legendary,
            {"stat": "crit_d", "amount": 0.2}
        )