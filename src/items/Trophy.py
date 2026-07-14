from src.items.Item import Item, ItemType, ItemRarity

class BossTrophy(Item):
    def __init__(self):
        super().__init__(
            name="trophée de boss",
            price=10000,
            description="Un trophée légendaire récompensant le premier joueur à vaincre le boss de fin du serveur.",
            item_type=ItemType.collectible,
            rarity=ItemRarity.unique
        )
