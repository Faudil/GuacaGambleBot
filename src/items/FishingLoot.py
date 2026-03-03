from src.items.Item import ItemRarity
from src.items.ResourceItem import ResourceItem


class OldBoot(ResourceItem):
    def __init__(self):
        super().__init__(
            "Vieille botte", 1, "Un caillou tout nul. Sert à rien.", ItemRarity.common,
            {"stat": "defense", "amount": 1} # Le cuir durcit un peu la peau
        )

class Trout(ResourceItem):
    def __init__(self):
        super().__init__(
            "Truite", 10, "Un poisson d'eau douce.", ItemRarity.common,
            {"stat": "max_hp", "amount": 2} # +2 PV Max
        )

class Salmon(ResourceItem):
    def __init__(self):
        super().__init__(
            "Saumon", 10, "Parfait pour faire des sushis.", ItemRarity.common,
            {"stat": "speed", "amount": 1} # +1 Vitesse (Un poisson qui remonte le courant)
        )

class Sardine(ResourceItem):
    def __init__(self):
        super().__init__(
            "Sardine", 15, "Un petit poisson d'eau de mer", ItemRarity.common,
            {"stat": "acc", "amount": 1} # +1 Précision (Les bancs de sardines sont vifs)
        )

class Carp(ResourceItem):
    def __init__(self):
        super().__init__(
            "Carpe", 25, "Le meilleur poisson d'eau douce.", ItemRarity.rare,
            {"stat": "max_hp", "amount": 3} # +3 PV Max
        )

class Pufferfish(ResourceItem):
    def __init__(self):
        super().__init__(
            "Poisson-Globe", 50, "Attention, ça pique !", ItemRarity.rare,
            {"stat": "defense", "amount": 2} # +2 Défense (Les épines !)
        )

class Swordfish(ResourceItem):
    def __init__(self):
        super().__init__(
            "Espadon", 150, "Un poisson combattant majestueux.", ItemRarity.epic,
            {"stat": "crit_c", "amount": 2}
        )

class Shark(ResourceItem):
    def __init__(self):
        super().__init__(
            "Requin", 100, "INCROYABLE ! Ça vaut une fortune !", ItemRarity.epic,
            {"stat": "crit_d", "amount": 0.1}
        )

class Whale(ResourceItem):
    def __init__(self):
        super().__init__(
            "Woaw", 300, "INCROYABLE ! Ça vaut une fortune !", ItemRarity.legendary,
            {"stat": "max_hp", "amount": 10}
        )

class KrakenTentacle(ResourceItem):
    def __init__(self):
        super().__init__(
            "Tentacule de Kraken", 500, "TU AS PÊCHÉ UN MONSTRE ?!", ItemRarity.legendary,
            {"stat": "atk", "amount": 3}
        )