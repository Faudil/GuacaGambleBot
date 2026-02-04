from src.items.Item import ItemRarity
from src.items.ResourceItem import ResourceItem


class Wheat(ResourceItem):
    def __init__(self):
        super().__init__("Blé", 5, "Indispensable pour faire du pain.", ItemRarity.common)

class Oat(ResourceItem):
    def __init__(self):
        super().__init__("Avoine", 8, "Parfait pour le petit déjeuner.", ItemRarity.common)

class Corn(ResourceItem):
    def __init__(self):
        super().__init__("Maïs", 12, "Fait aussi du pop-corn !", ItemRarity.common)

class Potato(ResourceItem):
    def __init__(self):
        super().__init__("Patate", 20, "On peut en faire de la vodka...", ItemRarity.rare)

class Tomato(ResourceItem):
    def __init__(self):
        super().__init__("Tomate", 25, "Un fruit ou un légume ? Le débat continue.", ItemRarity.rare)

class Pumpkin(ResourceItem):
    def __init__(self):
        super().__init__("Citrouille", 40, "Parfait pour Halloween.", ItemRarity.rare)

class CoffeeBean(ResourceItem):
    def __init__(self):
        super().__init__("Grain de Café", 60, "L'or noir du matin.", ItemRarity.epic)

class CocoaBean(ResourceItem):
    def __init__(self):
        super().__init__("Fève de Cacao", 75, "L'ingrédient principal du bonheur (chocolat).", ItemRarity.epic)

class Strawberry(ResourceItem):
    def __init__(self):
        super().__init__("Fraise", 90, "Rouge, sucrée et juteuse.", ItemRarity.epic)


class GoldenApple(ResourceItem):
    def __init__(self):
        super().__init__("Pomme Dorée", 250, "Elle brille d'une lueur magique.", ItemRarity.epic)

class StarFruit(ResourceItem):
    def __init__(self):
        super().__init__("Fruit Étoile", 500, "Un fruit cosmique d'une autre dimension.", ItemRarity.legendary)

# --- ÉCHEC ---
class RottenPlant(ResourceItem):
    def __init__(self):
        super().__init__("Plante Pourrie", 0, "Tu as mal géré ta ferme...", ItemRarity.common)