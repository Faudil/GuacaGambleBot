from src.items.Item import ItemRarity
from src.items.ResourceItem import ResourceItem

class BoneDust(ResourceItem):
    def __init__(self):
        super().__init__(
            "Poussière d'os", 1, "De la poussière d'os de fossile complètement détruit.", ItemRarity.common
        )

class BrokenFossil(ResourceItem):
    def __init__(self):
        super().__init__(
            "Fossile Abîmé", 50, "Un fossile mal extrait, il a perdu de sa valeur.", ItemRarity.common
        )

class CommonFossil(ResourceItem):
    def __init__(self):
        super().__init__(
            f"Fossile Commun", 150, f"Un fossile intact d'animal commun.", ItemRarity.rare
        )

class RareFossil(ResourceItem):
    def __init__(self):
        super().__init__(
            f"Fossile Rare", 300, f"Un fossile intact d'animal rare.", ItemRarity.rare
        )

class EpicFossil(ResourceItem):
    def __init__(self):
        super().__init__(
            f"Fossile Épique", 500, f"Un fossile intact d'animal épique.", ItemRarity.epic
        )

class LegendaryFragment(ResourceItem):
    def __init__(self):
        super().__init__(
            f"Fragment Légendaire", 1000, f"Un fragment légendaire d'un T-Rex !", ItemRarity.legendary
        )

class PureDNA(ResourceItem):
    def __init__(self):
        super().__init__(
            f"ADN Pur", 3000, f"De l'ADN de dinosaure parfaitement conservé. Incroyable !", ItemRarity.unique
        )
