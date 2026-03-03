from src.items.Item import Item, ItemRarity, ItemType


class MysteryEgg(Item):
    def __init__(self):
        super().__init__(
            "Œuf Mystère",
            10000,
            "Un œuf frémissant... Tape !hatch pour l'ouvrir ! (Peut contenir un familier légendaire)",
            ItemType.consumable,
            ItemRarity.epic,
        )