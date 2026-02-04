from src.items.Item import ItemRarity
from src.items.ResourceItem import ResourceItem


class OldBoot(ResourceItem):
    def __init__(self):
        super().__init__("Vieille botte", 1, "Un caillou tout nul. Sert à rien.", ItemRarity.common)

class Trout(ResourceItem):
    def __init__(self):
        super().__init__("Truite", 10, "Un poisson d'eau douce.", ItemRarity.common)

class Salmon(ResourceItem):
    def __init__(self):
        super().__init__("Saumon", 10, "Parfait pour faire des sushis.", ItemRarity.common)

class Pufferfish(ResourceItem):
    def __init__(self):
        super().__init__("Poisson-Globe", 50, "Attention, ça pique !", ItemRarity.rare)

class Swordfish(ResourceItem):
    def __init__(self):
        super().__init__("Espadon", 150, "Un poisson combattant majestueux.", ItemRarity.epic)

class KrakenTentacle(ResourceItem):
    def __init__(self):
        super().__init__("Tentacule de Kraken", 500, "TU AS PÊCHÉ UN MONSTRE ?!", ItemRarity.legendary)

class Sardine(ResourceItem):
    def __init__(self):
        super().__init__("Sardine", 15, "Un petit poisson d'eau de mer", ItemRarity.common)

class Carp(ResourceItem):
    def __init__(self):
        super().__init__("Carpe", 25, "Le meilleur poisson d'eau douce.", ItemRarity.rare)

class Shark(ResourceItem):
    def __init__(self):
        super().__init__("Requin", 100, "INCROYABLE ! Ça vaut une fortune !", ItemRarity.epic)

class Whale(ResourceItem):
    def __init__(self):
        super().__init__("Woaw", 300, "INCROYABLE ! Ça vaut une fortune !", ItemRarity.legendary)
