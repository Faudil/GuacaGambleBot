from src.items.Item import ItemRarity
from src.items.ResourceItem import ResourceItem


class Pebble(ResourceItem):
    def __init__(self):
        super().__init__(
            "Caillou", 1, "Un caillou tout nul. Sert à rien.", ItemRarity.common,
            {"stat": "max_hp", "amount": 1}
        )

class Coal(ResourceItem):
    def __init__(self):
        super().__init__(
            "Charbon", 5, "Pas mal pour se réchauffer.", ItemRarity.common,
            {"stat": "dge", "amount": 1}
        )

class IronOre(ResourceItem):
    def __init__(self):
        super().__init__(
            "Minerai de Fer", 10, "Utile pour forger des trucs solides.", ItemRarity.rare,
            {"stat": "defense", "amount": 1}
        )

class CopperOre(ResourceItem):
    def __init__(self):
        super().__init__(
            "Minerai de Cuivre", 15, "Utile pour forger des trucs solides.", ItemRarity.rare,
            {"stat": "speed", "amount": 1}
        )

class SilverOre(ResourceItem):
    def __init__(self):
        super().__init__(
            "Minerai d'argent", 25, "Utile pour forger des trucs solides.", ItemRarity.rare,
            {"stat": "acc", "amount": 1}
        )

class GoldNugget(ResourceItem):
    def __init__(self):
        super().__init__(
            "Pépite d'Or", 50, "Brillant ! Les marchands adorent ça.", ItemRarity.epic,
            {"stat": "crit_c", "amount": 1}
        )

class PlatinumOre(ResourceItem):
    def __init__(self):
        super().__init__(
            "Platine", 75, "Brillant ! Les marchands adorent ça.", ItemRarity.epic,
            {"stat": "defense", "amount": 2}
        )

class Emerald(ResourceItem):
    def __init__(self):
        super().__init__(
            "Emeraude", 100, "INCROYABLE ! Ça vaut une fortune !", ItemRarity.epic,
            {"stat": "dge", "amount": 2}
        )

class Diamond(ResourceItem):
    def __init__(self):
        super().__init__(
            "Diamant Brut", 300, "INCROYABLE ! Ça vaut une fortune !", ItemRarity.legendary,
            {"stat": "crit_d", "amount": 0.2}
        )